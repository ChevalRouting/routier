#!/usr/bin/env python3
import datetime
import html
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
from urllib.parse import urlsplit

USAGE = 'unknown command or wrong arguments, run scripts/repo.sh without arguments for the usage'

CHANNELS = ('nightly', 'stable')
ARCHES = ('x86_64', 'aarch64')
KEY = Path('/key/routier.rsa')
PACKAGES = ('routier', 'routier-openrc')
BROWSE = Path('/browse.html')
SITE_KEEP = ('repo',)
SETUP_STYLE = '  .setup { padding: 4px 16px; margin-bottom: 16px; }\n  .setup pre { overflow-x: auto; }\n'


class Failure(Exception):
    pass


class Entry:
    def __init__(self, stamp, size, name):
        self.stamp = stamp
        self.size = size
        self.name = name


class Stage:
    def __init__(self, channel, arch):
        self.channel = channel
        self.arch = arch
        self.remote_dir = target() / channel / arch
        self.dir = Path(tempfile.mkdtemp())
        self.remote = sorted(self.remote_dir.glob('*.apk')) if self.remote_dir.is_dir() else []
        if not self.remote:
            print(f'starting the {channel}/{arch} channel')

        for package in self.remote:
            shutil.copy2(package, self.dir / package.name)

    def packages(self):
        return sorted(self.dir.glob('*.apk'))

    def prune(self):
        days = keep_days(self.channel)
        if days <= 0:
            return

        cutoff = datetime.datetime.now(datetime.timezone.utc).timestamp() - days * 86400
        for package in self.remote:
            if package.stat().st_mtime < cutoff:
                (self.dir / package.name).unlink(missing_ok=True)
                print(f'pruned {package.name}')

    def index(self):
        index = self.dir / 'APKINDEX.tar.gz'
        index.unlink(missing_ok=True)
        packages = [p.name for p in self.packages()]
        if not packages:
            raise Failure('no packages left in the channel, nothing was published')

        result = subprocess.run(['apk', 'index', '--rewrite-arch', self.arch, '-o', index.name, *packages],
                                cwd=self.dir, capture_output=True, text=True)
        if result.returncode != 0:
            sys.stderr.write(result.stderr)
            raise Failure('apk index failed, nothing was published')

        sys.stderr.write(index_warnings(result.stderr))
        described = run('tar', '-Oxzf', str(index), 'APKINDEX').splitlines()
        got = sum(1 for line in described if line.startswith('P:'))
        if got != len(packages):
            raise Failure(f'index describes {got} of {len(packages)} packages, nothing was published')

        run('abuild-sign', '-q', str(index))
        if '.SIGN.RSA.routier.rsa.pub' not in run('tar', '-tzf', str(index)).splitlines():
            raise Failure('index signature missing, nothing was published')

    def push(self):
        self.remote_dir.mkdir(parents=True, exist_ok=True)
        for package in self.packages():
            if not (self.remote_dir / package.name).exists():
                publish(package, self.remote_dir / package.name)

        publish(KEY.with_suffix('.rsa.pub'), target() / 'routier.rsa.pub')
        publish(self.dir / 'APKINDEX.tar.gz', self.remote_dir / 'APKINDEX.tar.gz')
        for package in self.remote:
            if not (self.dir / package.name).exists():
                package.unlink(missing_ok=True)

        write_pages()
        shutil.rmtree(self.dir)


def target():
    return Path(os.environ.get('TARGET', '/target'))


def run(*args):
    result = subprocess.run(args, capture_output=True, text=True)
    if result.returncode != 0:
        sys.stderr.write(result.stderr)
        raise Failure(f'{args[0]} failed')

    return result.stdout


def publish(source, dest):
    dest.parent.mkdir(parents=True, exist_ok=True)
    partial = dest.with_name(f'.{dest.name}.partial')
    shutil.copyfile(source, partial)
    partial.chmod(0o644)
    os.replace(partial, dest)


def publish_text(text, dest):
    dest.parent.mkdir(parents=True, exist_ok=True)
    partial = dest.with_name(f'.{dest.name}.partial')
    partial.write_text(text)
    partial.chmod(0o644)
    os.replace(partial, dest)


def stamp(path):
    moment = datetime.datetime.fromtimestamp(path.stat().st_mtime, datetime.timezone.utc)
    return moment.strftime('%Y/%m/%d %H:%M:%S')


def listing(root):
    entries = []
    for path in sorted(root.rglob('*')):
        relative = path.relative_to(root)
        if path.is_file() and not any(part.startswith('.') for part in relative.parts):
            entries.append(Entry(stamp(path), path.stat().st_size, relative.as_posix()))

    return entries


def keep_days(channel):
    if os.environ.get('KEEP_DAYS'):
        return int(os.environ['KEEP_DAYS'])

    return 30 if channel == 'nightly' else 0


def index_warnings(stderr):
    kept = []
    in_provider = False
    for line in stderr.splitlines(keepends=True):
        if line.startswith('WARNING: No provider'):
            in_provider = True
            continue

        if in_provider and line[:1].isspace():
            continue

        in_provider = False
        if line.startswith('WARNING: Total of') and 'unsatisfiable' in line:
            continue

        if 'Your repository may be broken' in line:
            continue

        kept.append(line)

    return ''.join(kept)


def install_key():
    if not KEY.is_file():
        raise Failure('signing key missing, set SIGN_KEY')

    abuild = Path('/root/.abuild')
    abuild.mkdir(parents=True, exist_ok=True)
    shutil.copy(KEY, abuild / 'routier.rsa')
    (abuild / 'routier.rsa').chmod(0o600)
    shutil.copy(KEY.with_suffix('.rsa.pub'), abuild / 'routier.rsa.pub')
    (abuild / 'abuild.conf').write_text(f'PACKAGER_PRIVKEY="{abuild}/routier.rsa"\n')
    shutil.copy(KEY.with_suffix('.rsa.pub'), '/etc/apk/keys/routier.rsa.pub')


def human(size):
    for unit, scale in (('GiB', 1 << 30), ('MiB', 1 << 20), ('KiB', 1 << 10)):
        if size >= scale:
            return f'{size / scale:.1f} {unit}'

    return f'{size} B'


def row(kind, href, label, size='', date=''):
    icon = 'up' if href == '..' else 'folder' if kind == 'dir' else 'file'
    return (f'      <a class="row {kind}" href="{html.escape(href)}">\n'
            f'        <svg class="icon"><use href="#i-{icon}"/></svg>\n'
            f'        <span class="name">{html.escape(label)}</span>\n'
            f'        <span class="meta size">{size}</span>\n'
            f'        <span class="meta date">{date}</span>\n'
            '      </a>\n')


def setup_note(repo, site):
    repo, site = html.escape(repo), html.escape(site)
    return ('    <div class="boxed-list setup">\n'
            f'      <p>Add the repository to an Alpine or Routier host, then <code>apk add routier</code>. '
            f'Documentation: <a href="{site}">{site}</a>.</p>\n'
            f'      <pre>wget -qO /etc/apk/keys/routier.rsa.pub {repo}/routier.rsa.pub\n'
            f'echo {repo}/nightly &gt;&gt; /etc/apk/repositories</pre>\n'
            '    </div>\n')


def page_head(template, title):
    lines = []
    for line in template.splitlines(keepends=True):
        if line.startswith('</style>'):
            lines.append(SETUP_STYLE)

        lines.append(line.replace('{{html .Name}}', html.escape(title)))
        if '<span class="wordmark">' in line:
            break

    return ''.join(lines) + '    </div>\n'


def tree_entries(entries):
    kept = [e for e in entries if not e.name.startswith('state/') and e.name.split('/')[-1] != 'index.html']
    return sorted(kept, key=lambda e: e.name)


def render_pages(entries, template, repo, site):
    base = urlsplit(repo).path.rstrip('/')
    subdirs = {'': []}
    files = {'': []}
    for entry in entries:
        parts = entry.name.split('/')
        path = ''
        for part in parts[:-1]:
            child = f'{path}{part}/'
            if child not in subdirs:
                subdirs[child] = []
                files[child] = []
                subdirs[path].append(part)

            path = child

        files[path].append((parts[-1], entry))

    pages = {}
    for path in subdirs:
        rows = [row('dir', '..', 'Parent directory')] if path else []
        rows += [row('dir', f'{base}/{path}{name}/', name) for name in subdirs[path]]
        rows += [row('file', f'{base}/{entry.name}', name, human(entry.size), entry.stamp[:16].replace('/', '-'))
                 for name, entry in files[path]]
        title = f'Routier packages – /{path}' if path else 'Routier packages'
        pages[path] = (page_head(template, title)
                       + (breadcrumbs(base, path) if path else '')
                       + '  </header>\n\n  <main>\n'
                       + ('' if path else setup_note(repo, site))
                       + '    <div class="boxed-list">\n' + ''.join(rows) + '    </div>\n'
                       + f'    <p class="summary">{len(subdirs[path])} directories, {len(files[path])} files</p>\n'
                       + '  </main>\n</body>\n</html>\n')

    return pages


def breadcrumbs(base, path):
    links = [f'      <a href="{html.escape(base)}/">Home</a>\n']
    link = f'{base}/'
    for segment in path.rstrip('/').split('/'):
        link = f'{link}{segment}/'
        links.append(f'      <a href="{html.escape(link)}">{html.escape(segment)}</a>\n')

    return '    <nav class="breadcrumbs">\n' + ''.join(links) + '    </nav>\n'


def write_pages():
    entries = tree_entries(listing(target()))
    pages = render_pages(entries, BROWSE.read_text(), os.environ['REPO_URL'], os.environ['SITE_URL'])
    for path, body in pages.items():
        publish_text(body, target() / path / 'index.html')


def mirror(source, dest, keep=()):
    wanted = set()
    for path in sorted(source.rglob('*')):
        relative = path.relative_to(source)
        if relative.parts[0] in keep:
            raise Failure(f'{relative} would overwrite the reserved {relative.parts[0]}/ tree')

        wanted.add(relative)
        if path.is_dir():
            (dest / relative).mkdir(parents=True, exist_ok=True)
        else:
            publish(path, dest / relative)

    for path in sorted(dest.rglob('*'), reverse=True):
        relative = path.relative_to(dest)
        if relative.parts[0] in keep or relative in wanted:
            continue

        if not path.is_dir():
            path.unlink()
        elif not any(path.iterdir()):
            path.rmdir()


def check_channel(channel):
    if channel not in CHANNELS:
        raise Failure('channel must be nightly or stable')


def check_arch(arch):
    if arch not in ARCHES:
        raise Failure(f'unsupported arch {arch}')


def cmd_add(channel, arch, version, release):
    check_channel(channel)
    check_arch(arch)
    install_key()
    stage = Stage(channel, arch)
    for package in PACKAGES:
        name = f'{package}-{version}-r{release}.apk'
        source = Path('/pkgs') / name
        if not source.is_file():
            raise Failure(f'missing dist/{arch}/{name}')

        shutil.copy(source, stage.dir / name)

    stage.prune()
    stage.index()
    stage.push()
    print(f'published routier {version}-r{release} to {channel}/{arch}')


def cmd_remove(channel, arch, build):
    check_channel(channel)
    check_arch(arch)
    install_key()
    stage = Stage(channel, arch)
    found = False
    for package in PACKAGES:
        path = stage.dir / f'{package}-{build}.apk'
        if path.exists():
            path.unlink()
            found = True

    if not found:
        raise Failure(f'routier {build} is not in {channel}/{arch}')

    stage.index()
    stage.push()
    print(f'removed routier {build} from {channel}/{arch}')


def cmd_promote(arch, build):
    check_arch(arch)
    install_key()
    stage = Stage('stable', arch)
    for package in PACKAGES:
        source = target() / 'nightly' / arch / f'{package}-{build}.apk'
        if not source.is_file():
            raise Failure(f'routier {build} is not in nightly/{arch}')

        shutil.copy(source, stage.dir / source.name)

    stage.index()
    stage.push()
    print(f'promoted routier {build} to stable/{arch}')


def cmd_prune(channel, arch):
    check_channel(channel)
    check_arch(arch)
    install_key()
    stage = Stage(channel, arch)
    stage.prune()
    stage.index()
    stage.push()


def cmd_reindex(channel, arch):
    check_channel(channel)
    check_arch(arch)
    install_key()
    stage = Stage(channel, arch)
    stage.index()
    stage.push()
    print(f'reindexed {channel}/{arch}')


def cmd_list():
    for entry in listing(target()):
        if entry.name.endswith(('.apk', '.iso')):
            print(f'{entry.stamp} {entry.size:>12} {entry.name}')


def cmd_iso(arch):
    check_arch(arch)
    images = sorted(Path('/iso').glob(f'*-{arch}.iso'), key=lambda p: p.stat().st_mtime, reverse=True)
    if not images:
        raise Failure(f'no dist/iso image for {arch}')

    name = images[0].name
    iso_dir = target() / 'iso'
    publish(images[0], iso_dir / name)
    publish(iso_dir / name, iso_dir / f'routier-latest-{arch}.iso')
    published = [p for p in iso_dir.glob(f'*-{arch}.iso') if not p.name.startswith('routier-latest-')]
    published.sort(key=lambda p: p.stat().st_mtime, reverse=True)
    for old in published[int(os.environ.get('ISO_KEEP') or 5):]:
        old.unlink()
        print(f'pruned {old.name}')

    write_pages()
    print(f'published {name} as routier-latest-{arch}.iso')


def cmd_page():
    write_pages()
    print(f'rewrote the listing pages of {target()}')


def cmd_site():
    if not Path('/site/index.html').is_file():
        raise Failure('website/build is empty, build the site first')

    mirror(Path('/site'), target(), SITE_KEEP)
    print('published the website')


def cmd_counter(action, arch, version, release=None):
    state = target() / 'state' / 'releases' / arch / version
    if action == 'next' and release is None:
        last = state.read_text().strip() if state.is_file() else ''
        print(int(last) + 1 if last else 0)
    elif action == 'set' and release is not None:
        publish_text(f'{int(release)}\n', state)
    else:
        raise Failure(USAGE)


COMMANDS = {
    'add': (cmd_add, 4, 4),
    'remove': (cmd_remove, 3, 3),
    'promote': (cmd_promote, 2, 2),
    'prune': (cmd_prune, 2, 2),
    'reindex': (cmd_reindex, 2, 2),
    'list': (cmd_list, 0, 0),
    'iso': (cmd_iso, 1, 1),
    'counter': (cmd_counter, 3, 4),
    'site': (cmd_site, 0, 0),
    'page': (cmd_page, 0, 0),
}


def main(args):
    if not args or args[0] not in COMMANDS:
        raise Failure(USAGE)

    handler, least, most = COMMANDS[args[0]]
    if not least <= len(args) - 1 <= most:
        raise Failure(USAGE)

    handler(*args[1:])


if __name__ == '__main__':
    try:
        main(sys.argv[1:])
    except Failure as failure:
        print(failure, file=sys.stderr)
        sys.exit(1)
