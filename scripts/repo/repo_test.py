#!/usr/bin/env python3
import os
from pathlib import Path
import tempfile
import unittest

import repo

TEMPLATE = (Path(__file__).resolve().parents[2] / 'assets' / 'browse-routier.html').read_text()
FILES = {
    'routier.rsa.pub': 800,
    'iso/routier-installer-0.0.5-r500-x86_64.iso': 3000,
    'index.html': 10,
    'nightly/index.html': 10,
    'nightly/x86_64/APKINDEX.tar.gz': 2048,
    'nightly/x86_64/routier-0.0.5-r500.apk': 5242880,
    'nightly/x86_64/.routier-0.0.5-r501.apk.partial': 1,
    'state/releases/x86_64/0.0.5': 2,
}
MTIME = 1795430460


def make_tree(root, files):
    for name, size in files.items():
        path = root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        with open(path, 'wb') as f:
            f.truncate(size)
        os.utime(path, (MTIME, MTIME))


class TreeTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        make_tree(self.root, FILES)

    def tearDown(self):
        self.tmp.cleanup()

    def pages(self, repo_url='https://routier.chevalrouting.org/repo'):
        entries = repo.tree_entries(repo.listing(self.root))
        return repo.render_pages(entries, TEMPLATE, repo_url, 'https://routier.chevalrouting.org')

    def test_listing_skips_hidden_files(self):
        names = [e.name for e in repo.listing(self.root)]
        self.assertIn('nightly/x86_64/routier-0.0.5-r500.apk', names)
        self.assertNotIn('nightly/x86_64/.routier-0.0.5-r501.apk.partial', names)
        entry = next(e for e in repo.listing(self.root) if e.name == 'routier.rsa.pub')
        self.assertEqual(entry.size, 800)
        self.assertEqual(entry.stamp, '2026/11/23 10:41:00')

    def test_tree_drops_state_and_pages(self):
        names = [e.name for e in repo.tree_entries(repo.listing(self.root))]
        self.assertNotIn('index.html', names)
        self.assertNotIn('nightly/index.html', names)
        self.assertFalse(any(n.startswith('state/') for n in names))
        self.assertEqual(names, sorted(names))

    def test_every_directory_has_a_page(self):
        self.assertEqual(sorted(self.pages()), ['', 'iso/', 'nightly/', 'nightly/x86_64/'])

    def test_links_stay_under_the_repository_path(self):
        root = self.pages()['']
        self.assertIn('href="/repo/iso/"', root)
        self.assertIn('href="/repo/nightly/"', root)
        self.assertIn('href="/repo/routier.rsa.pub"', root)
        self.assertNotIn('href="/iso/"', root)
        leaf = self.pages()['nightly/x86_64/']
        self.assertIn('href="/repo/nightly/x86_64/routier-0.0.5-r500.apk"', leaf)
        self.assertIn('<a href="/repo/">Home</a>', leaf)
        self.assertIn('<a href="/repo/nightly/">nightly</a>', leaf)
        self.assertIn('<a href="/repo/nightly/x86_64/">x86_64</a>', leaf)

    def test_root_page_has_setup_and_counts(self):
        root = self.pages()['']
        self.assertIn('echo https://routier.chevalrouting.org/repo/nightly &gt;&gt; /etc/apk/repositories', root)
        self.assertIn('2 directories, 1 files', root)
        self.assertIn('<title>Routier packages</title>', root)
        self.assertIn('.setup pre', root)
        self.assertNotIn('Parent directory', root)
        self.assertTrue(root.endswith('</html>\n'))

    def test_file_rows_show_size_and_date(self):
        leaf = self.pages()['nightly/x86_64/']
        self.assertIn('5.0 MiB', leaf)
        self.assertIn('2026-11-23 10:41', leaf)
        self.assertIn('Parent directory', leaf)
        self.assertIn('<title>Routier packages – /nightly/x86_64/</title>', leaf)

    def test_repository_at_the_domain_root(self):
        self.assertIn('href="/iso/"', self.pages('https://repo.example.org')[''])

    def test_counter(self):
        os.environ['TARGET'] = str(self.root)
        try:
            repo.cmd_counter('set', 'aarch64', '0.0.6', '4')
            self.assertEqual((self.root / 'state/releases/aarch64/0.0.6').read_text(), '4\n')
        finally:
            del os.environ['TARGET']


class MirrorTest(unittest.TestCase):
    def test_mirror_replaces_the_site_and_keeps_the_repository(self):
        with tempfile.TemporaryDirectory() as tmp:
            source, dest = Path(tmp) / 'build', Path(tmp) / 'public'
            make_tree(source, {'index.html': 5, 'docs/intro/index.html': 5})
            make_tree(dest, {'index.html': 1, 'old/page.html': 1, 'repo/routier.rsa.pub': 800})
            repo.mirror(source, dest, repo.SITE_KEEP)
            names = sorted(p.relative_to(dest).as_posix() for p in dest.rglob('*') if p.is_file())
            self.assertEqual(names, ['docs/intro/index.html', 'index.html', 'repo/routier.rsa.pub'])
            self.assertFalse((dest / 'old').exists())
            self.assertEqual((dest / 'index.html').stat().st_size, 5)

    def test_mirror_refuses_to_overwrite_the_repository(self):
        with tempfile.TemporaryDirectory() as tmp:
            source, dest = Path(tmp) / 'build', Path(tmp) / 'public'
            make_tree(source, {'repo/index.html': 5})
            dest.mkdir()
            with self.assertRaises(repo.Failure):
                repo.mirror(source, dest, repo.SITE_KEEP)


class HelpersTest(unittest.TestCase):
    def test_human(self):
        self.assertEqual(repo.human(512), '512 B')
        self.assertEqual(repo.human(2048), '2.0 KiB')
        self.assertEqual(repo.human(3 << 30), '3.0 GiB')

    def test_index_warnings_drop_known_noise(self):
        stderr = ('WARNING: No provider for dependency foo\n'
                  '  required by: routier\n'
                  'WARNING: Total of 1 unsatisfiable package names. Your repository may be broken.\n'
                  'something else\n')
        self.assertEqual(repo.index_warnings(stderr), 'something else\n')

    def test_iso_names(self):
        self.assertEqual(repo.iso_names('x86_64', 'installer'),
                         ('routier-installer-[0-9]*-x86_64.iso', 'routier-latest-x86_64.iso'))
        self.assertEqual(repo.iso_names('aarch64', 'installer-virt'),
                         ('routier-installer-virt-[0-9]*-aarch64.iso', 'routier-latest-installer-virt-aarch64.iso'))
        self.assertEqual(repo.iso_names('x86_64', 'lts')[0], 'routier-[0-9]*-x86_64.iso')

    def test_cloud_and_iso_images_publish_with_latest_names(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            make_tree(root / 'images', {'cloud/routier-cloud-0.0.5-r502-x86_64.qcow2': 7,
                                        'iso/routier-installer-virt-0.0.5-r502-x86_64.iso': 5})
            env = {'TARGET': str(root / 'public'), 'IMAGES': str(root / 'images'),
                   'REPO_URL': 'https://x.org/repo', 'SITE_URL': 'https://x.org'}
            os.environ.update(env)
            browse = repo.BROWSE
            repo.BROWSE = Path(__file__).resolve().parents[2] / 'assets' / 'browse-routier.html'
            try:
                repo.cmd_cloud('x86_64')
                repo.cmd_iso('x86_64', 'installer-virt')
            finally:
                repo.BROWSE = browse
                for key in env:
                    del os.environ[key]
            names = sorted(p.relative_to(root / 'public').as_posix() for p in (root / 'public').rglob('*.*'))
            self.assertIn('cloud/routier-latest-cloud-x86_64.qcow2', names)
            self.assertIn('iso/routier-latest-installer-virt-x86_64.iso', names)
            self.assertIn('cloud/index.html', names)

    def test_iso_flavors_do_not_overlap(self):
        names = ['routier-installer-0.0.5-r502-x86_64.iso', 'routier-installer-virt-0.0.5-r502-x86_64.iso']
        for flavor, expected in (('installer', names[0]), ('installer-virt', names[1])):
            pattern = repo.iso_names('x86_64', flavor)[0]
            self.assertEqual([n for n in names if Path(n).match(pattern)], [expected])

    def test_keep_days(self):
        self.assertEqual(repo.keep_days('nightly'), 30)
        self.assertEqual(repo.keep_days('stable'), 0)


if __name__ == '__main__':
    unittest.main()
