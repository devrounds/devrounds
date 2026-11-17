"""Download, verify, cache and run the devrounds binary."""
import hashlib
import io
import os
import platform
import subprocess
import sys
import tarfile
import urllib.request
import zipfile
from pathlib import Path

from . import __version__

BASE = "https://github.com/devrounds/devrounds/releases/download/v" + __version__


def _target():
    system = {"Linux": "linux", "Darwin": "darwin", "Windows": "windows"}.get(platform.system())
    arch = {"x86_64": "amd64", "amd64": "amd64", "arm64": "arm64", "aarch64": "arm64"}.get(
        platform.machine().lower()
    )
    if not system or not arch:
        raise RuntimeError("unsupported platform %s/%s" % (platform.system(), platform.machine()))
    return system, arch


def _cache_dir():
    if os.name == "nt":
        base = os.environ.get("LOCALAPPDATA") or str(Path.home() / "AppData" / "Local")
    elif sys.platform == "darwin":
        base = str(Path.home() / "Library" / "Caches")
    else:
        base = os.environ.get("XDG_CACHE_HOME") or str(Path.home() / ".cache")
    return Path(base) / "devrounds" / __version__


def _fetch(url):
    with urllib.request.urlopen(url, timeout=60) as resp:
        return resp.read()


def _ensure_binary():
    system, arch = _target()
    name = "devrounds.exe" if system == "windows" else "devrounds"
    exe = _cache_dir() / name
    if exe.exists():
        return exe

    ext = "zip" if system == "windows" else "tar.gz"
    archive = "devrounds_%s_%s_%s.%s" % (__version__, system, arch, ext)
    sys.stderr.write("devrounds: downloading %s\n" % archive)
    data = _fetch(BASE + "/" + archive)
    sums = _fetch(BASE + "/checksums.txt").decode("utf-8")

    expected = None
    for line in sums.splitlines():
        parts = line.split()
        if len(parts) == 2 and parts[1] == archive:
            expected = parts[0]
    if expected is None:
        raise RuntimeError("no checksum listed for " + archive)
    if hashlib.sha256(data).hexdigest() != expected:
        raise RuntimeError("checksum mismatch for " + archive)

    if ext == "zip":
        with zipfile.ZipFile(io.BytesIO(data)) as zf:
            binary = zf.read(name)
    else:
        with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as tf:
            member = tf.extractfile(name)
            if member is None:
                raise RuntimeError(name + " not found in archive")
            binary = member.read()

    exe.parent.mkdir(parents=True, exist_ok=True)
    tmp = exe.with_name(name + ".part%d" % os.getpid())
    tmp.write_bytes(binary)
    if system != "windows":
        tmp.chmod(0o755)
    os.replace(str(tmp), str(exe))
    return exe


def main():
    try:
        exe = _ensure_binary()
    except Exception as err:
        sys.stderr.write("devrounds: %s\n" % err)
        sys.exit(1)
    sys.exit(subprocess.call([str(exe)] + sys.argv[1:]))
