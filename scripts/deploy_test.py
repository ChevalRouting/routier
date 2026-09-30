#!/usr/bin/env python3
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SCRIPTS = Path(__file__).resolve().parent
MOCK = '''#!/usr/bin/env python3
import json, os, pathlib, sys
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ['CALL_LOG'], 'a') as f:
    f.write(json.dumps([name, *args]) + '\\n')
if os.environ.get('FAIL_COMMAND') == name:
    sys.exit(7)
if name == 'ssh':
    command = args[-1]
    if command == 'apk --print-arch':
        print(os.environ.get('HOST_ARCH', 'x86_64'))
    elif command.startswith('mktemp'):
        print(os.environ.get('STAGING', '/tmp/routier-deploy.ABC12345'))
    elif 'deploy-install.sh' in command and os.environ.get('FAIL_INSTALL'):
        sys.exit(9)
elif name == 'task':
    root = pathlib.Path(args[args.index('--dir') + 1])
    values = dict(a.split('=', 1) for a in args if '=' in a)
    dest = root / 'dist' / values['ARCH']
    dest.mkdir(parents=True, exist_ok=True)
    for package in ['routier', 'routier-openrc']:
        (dest / (package + '-' + values['VERSION'] + '-r' + values['RELEASE'] + '.apk')).touch()
    (root / 'routier.rsa.pub').write_text('public key')
elif name == 'id':
    print(os.environ.get('MOCK_UID', '0'))
elif name == 'doas':
    os.environ['MOCK_UID'] = '0'
    os.execvp(args[0], args)
elif name == 'apk':
    state = pathlib.Path(os.environ['INSTALL_STATE'])
    if args == ['--print-arch']:
        print(os.environ.get('HOST_ARCH', 'x86_64'))
    elif args[:2] == ['info', '-e']:
        package, expected = args[2].split('=', 1)
        version = '1.2.3-r4' if state.exists() or os.environ.get('ALREADY_INSTALLED') else '1.2.2-r0'
        if state.exists() and os.environ.get('WRONG_PACKAGE') == package:
            version = '1.2.3-r3'
        if state.exists() and os.environ.get('MISSING_PACKAGE') == package:
            sys.exit(1)
        if version != expected:
            sys.exit(1)
        print(package)
    elif args[:2] == ['info', '-v']:
        print(args[2] + ': Declarative router/firewall manager')
    elif args[0] == 'add':
        if os.environ.get('FAIL_APK_ADD'):
            sys.exit(7)
        state.touch()
elif name == 'routier':
    print('1.2.3-r4')
'''


class DeployTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'scripts').mkdir()
        for name in ['deploy.sh', 'deploy-install.sh']:
            shutil.copy(SCRIPTS / name, self.root / 'scripts' / name)
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        for name in ['ssh', 'scp', 'task', 'apk', 'id', 'install', 'routier', 'doas']:
            path = self.bin / name
            path.write_text(MOCK)
            path.chmod(0o755)
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ['PATH'],
                        CALL_LOG=str(self.root / 'calls'), INSTALL_STATE=str(self.root / 'installed'),
                        TASK_BIN=str(self.bin / 'task'))

    def run_script(self, script='deploy.sh', args=None, **env):
        return subprocess.run(['sh', str(self.root / 'scripts' / script),
                               *(args or ['admin@router', '1.2.3', '4'])],
                              env=dict(self.env, **env), capture_output=True, text=True)

    def calls(self):
        path = self.root / 'calls'
        return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []

    def test_builds_remote_arch_uploads_only_public_key_and_cleans_up(self):
        result = self.run_script(HOST_ARCH='aarch64')
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self.calls()
        build = next(c for c in calls if c[0] == 'task')
        self.assertIn('ARCH=aarch64', build)
        self.assertIn('VERSION=1.2.3', build)
        self.assertIn('RELEASE=4', build)
        uploads = [c for c in calls if c[0] == 'scp']
        self.assertEqual(len(uploads), 3)
        self.assertIn(str(self.root / 'routier.rsa.pub'), uploads[-1])
        self.assertNotIn(str(self.root / 'routier.rsa'), sum(uploads, []))
        self.assertIn("rm -rf -- '/tmp/routier-deploy.ABC12345'", calls[-1])

    def seed_packages(self, arch='x86_64', version='1.2.3', release='4'):
        dest = self.root / 'dist' / arch
        dest.mkdir(parents=True, exist_ok=True)
        for package in ['routier', 'routier-openrc']:
            (dest / f'{package}-{version}-r{release}.apk').touch()
        (self.root / 'routier.rsa.pub').write_text('public key')
        return dest

    def test_matching_packages_skip_build(self):
        self.seed_packages()
        result = self.run_script(FAIL_COMMAND='task')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(any(c[0] == 'task' for c in self.calls()))
        self.assertEqual(len([c for c in self.calls() if c[0] == 'scp']), 3)

    def test_either_missing_package_triggers_build(self):
        for package in ['routier', 'routier-openrc']:
            dest = self.seed_packages()
            (dest / f'{package}-1.2.3-r4.apk').unlink()
            before = len([c for c in self.calls() if c[0] == 'task'])
            result = self.run_script()
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(len([c for c in self.calls() if c[0] == 'task']), before + 1)

    def test_other_versions_releases_and_architectures_do_not_skip_build(self):
        self.seed_packages(version='1.2.2')
        self.seed_packages(release='3')
        self.seed_packages(arch='aarch64')
        result = self.run_script()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(any(c[0] == 'task' for c in self.calls()))

    def test_rejects_unsafe_host_before_ssh(self):
        for host in ['-oProxyCommand=bad', 'router;echo bad', 'router$(bad)']:
            self.assertNotEqual(self.run_script(args=[host, '1.2.3', '4']).returncode, 0)
        self.assertEqual(self.calls(), [])

    def test_rejects_unsupported_arch_before_build(self):
        self.assertNotEqual(self.run_script(HOST_ARCH='armv7').returncode, 0)
        self.assertEqual(len(self.calls()), 1)

    def test_build_failure_stops_upload(self):
        self.assertNotEqual(self.run_script(FAIL_COMMAND='task').returncode, 0)
        self.assertFalse(any(c[0] == 'scp' for c in self.calls()))

    def test_rejects_unsafe_staging_path(self):
        self.assertNotEqual(self.run_script(STAGING='/tmp/routier-deploy.x/../../etc').returncode, 0)
        self.assertFalse(any(c[0] == 'scp' for c in self.calls()))

    def test_failed_install_retains_artifacts(self):
        result = self.run_script(FAIL_INSTALL='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('staged files remain', result.stderr)
        self.assertFalse(any('rm -rf' in c[-1] for c in self.calls()))

    def test_installer_elevates_and_verifies_both_packages(self):
        result = self.run_script('deploy-install.sh', ['x86_64', '1.2.3-r4'], MOCK_UID='1000')
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self.calls()
        self.assertTrue(any(c[0] == 'doas' for c in calls))
        self.assertIn(['apk', 'add', '--upgrade', './routier.apk', './routier-openrc.apk'], calls)
        self.assertIn(['apk', 'info', '-e', 'routier-openrc=1.2.3-r4'], calls)
        self.assertEqual(calls[-1], ['routier', 'version'])

    def test_same_version_or_arch_mismatch_does_not_mutate_host(self):
        for options in [{'ALREADY_INSTALLED': '1'}, {'HOST_ARCH': 'aarch64'}]:
            result = self.run_script('deploy-install.sh', ['x86_64', '1.2.3-r4'], **options)
            self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(c[0] == 'install' or c[:2] == ['apk', 'add'] for c in self.calls()))

    def test_install_verification_rejects_wrong_release_or_missing_package(self):
        for package in ['routier', 'routier-openrc']:
            for failure in ['WRONG_PACKAGE', 'MISSING_PACKAGE']:
                with self.subTest(package=package, failure=failure):
                    Path(self.env['INSTALL_STATE']).unlink(missing_ok=True)
                    (self.root / 'calls').unlink(missing_ok=True)
                    result = self.run_script('deploy-install.sh', ['x86_64', '1.2.3-r4'],
                                             **{failure: package})
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn(f'Expected {package}-1.2.3-r4', result.stderr)
                    self.assertFalse(any(c[0] == 'routier' for c in self.calls()))

    def test_package_failure_is_propagated(self):
        result = self.run_script('deploy-install.sh', ['x86_64', '1.2.3-r4'], FAIL_APK_ADD='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(any(c[:2] == ['apk', 'add'] for c in self.calls()))
        self.assertFalse(any(c[0] == 'routier' for c in self.calls()))


if __name__ == '__main__':
    unittest.main()
