_routier_host="$(hostname -f 2>/dev/null || hostname)"

_routier_ver() {
	[ -r /etc/routier/version ] || return 0
	read -r _v < /etc/routier/version
	[ -n "${_v:-}" ] && printf '|%s' "$_v"
}

if [ "$(id -u)" -eq 0 ]; then
	_routier_color='\[\033[1;31m\]'
	_routier_char='#'
else
	_routier_color='\[\033[1;32m\]'
	_routier_char='$'
fi

PS1="${_routier_color}\u\[\033[0m\]@\[\033[1;36m\]${_routier_host}\[\033[0m\033[2m\]\$(_routier_ver)\[\033[0m\]:\[\033[1;34m\]\w\[\033[0m\]${_routier_color}${_routier_char}\[\033[0m\] "
export PS1

unset _routier_host _routier_color _routier_char
