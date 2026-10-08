set -l engines rancher docker podman off
complete -c dock -f
complete -c dock -n "not __fish_seen_subcommand_from $engines" -a rancher -d 'Rancher Desktop, at ~/.rd/docker.sock'
complete -c dock -n "not __fish_seen_subcommand_from $engines" -a docker -d 'Docker Desktop, at ~/.docker/run/docker.sock'
complete -c dock -n "not __fish_seen_subcommand_from $engines" -a podman -d "Podman's default machine"
complete -c dock -n "not __fish_seen_subcommand_from $engines" -a off -d "docker's own current context"
complete -c dock -s d -l default -d 'New shells start on it too'
complete -c dock -s h -l help -d 'Show usage'
