#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"

path_bin="$(command -v no-mistakes || true)"
if [[ -z "${path_bin}" ]]; then
  install_bin="${HOME}/.no-mistakes/bin/no-mistakes"
else
  install_bin="${path_bin}"
  if [[ -L "${path_bin}" ]]; then
    if command -v realpath >/dev/null 2>&1; then
      install_bin="$(realpath "${path_bin}")"
    else
      link_target="$(readlink "${path_bin}")"
      if [[ "${link_target}" = /* ]]; then
        install_bin="${link_target}"
      else
        install_bin="$(cd -- "$(dirname -- "${path_bin}")" && pwd)/${link_target}"
      fi
    fi
  fi
fi

echo "Installing no-mistakes to ${install_bin}"
make -C "${repo_root}" build
mkdir -p "$(dirname -- "${install_bin}")"
install -m 755 "${repo_root}/bin/no-mistakes" "${install_bin}"

if ! "${install_bin}" daemon stop; then
  echo "daemon stop returned non-zero; continuing with daemon start"
fi
"${install_bin}" daemon start

echo
echo "Installed binary:"
"${install_bin}" --version

resolved_path="$(command -v no-mistakes || true)"
if [[ -n "${resolved_path}" ]]; then
  echo "PATH resolves no-mistakes to ${resolved_path}"
fi
