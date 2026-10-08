function dock --description 'Point this shell at Rancher Desktop, Docker Desktop or Podman'
    argparse --name=dock --max-args=1 d/default h/help -- $argv; or return

    if set -q _flag_help
        echo 'Usage: dock [-d | --default] [rancher | docker | podman | off]

Points docker, docker compose and lazydocker in this shell at one container
engine by setting DOCKER_HOST, and starts the engine if it isn\'t running.

  dock               pick one with fzf (a list, when not in a terminal)
  dock podman        this shell uses Podman
  dock off           this shell uses docker\'s own current context again
  dock -d rancher    new shells start on Rancher Desktop too
  dock -d            new shells start on whatever this shell uses'
        return
    end

    set -l engine $argv[1]
    if test -z "$engine"; and not set -q _flag_default
        set -l rows (_dock_list)
        if not isatty stdout; or not command -q fzf
            printf '%s\n' $rows
            return
        end
        set -l current 1
        for i in (seq (count $rows))
            string match -q '*this shell*' -- $rows[$i]; and set current $i
        end
        set -l row (printf '%s\n' $rows | fzf --height=~10 --layout=reverse --no-sort \
            --prompt='dock> ' --header='enter: use it in this shell, starting it if needed' \
            --bind="load:pos($current)"); or return
        set engine (string split -f1 ' ' -- $row)
    end

    set -l result 0
    switch "$engine"
        case ''
            # `dock --default` on its own: keep this shell's engine.
        case off
            set -e DOCKER_HOST
        case rancher docker podman
            set -l sock (_dock_socket $engine)
            if test -z "$sock"
                if command -q podman
                    echo 'dock: Podman has no machine yet; create one with `podman machine init`' >&2
                else
                    echo "dock: Podman isn't installed (brew install podman)" >&2
                end
                return 1
            end
            # A DOCKER_CONTEXT left over from elsewhere would hide DOCKER_HOST
            # from some tools and not others.
            set -e DOCKER_CONTEXT
            set -gx DOCKER_HOST unix://$sock
            _dock_up $sock; or _dock_start $engine $sock; or set result 1
        case '*'
            echo "dock: unknown engine '$engine', expected rancher, docker, podman or off" >&2
            return 1
    end

    # Universal, so it holds for every new shell on this machine; config.fish
    # applies it to shells that don't inherit a DOCKER_HOST.
    if set -q _flag_default
        if set -q DOCKER_HOST
            set -U dock_default $DOCKER_HOST
            echo "dock: new shells use $DOCKER_HOST"
        else
            set -Ue dock_default
            echo "dock: new shells use docker's own current context"
        end
    end
    return $result
end

# The socket each engine's Docker API listens on. Rancher Desktop's is in ~/.rd
# while it runs without administrative access, and Podman's is in $TMPDIR, so
# podman reports it.
function _dock_socket --argument-names engine
    switch $engine
        case rancher
            echo ~/.rd/docker.sock
        case docker
            echo ~/.docker/run/docker.sock
        case podman
            command -q podman; or return 1
            podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}' 2>/dev/null
    end
end

function _dock_title --argument-names engine
    switch $engine
        case rancher
            echo 'Rancher Desktop'
        case docker
            echo 'Docker Desktop'
        case podman
            echo Podman
    end
end

function _dock_up --argument-names sock
    string match -q OK -- (curl -s -m 2 --unix-socket $sock http://localhost/_ping)
end

function _dock_start --argument-names engine sock
    set -l title (_dock_title $engine)
    set -l start (date +%s)
    echo "dock: starting $title…" >&2
    switch $engine
        case rancher
            open -ga 'Rancher Desktop'
        case docker
            open -ga Docker
        case podman
            podman machine start >/dev/null
    end
    or return

    # The apps return at once, and their VM takes a minute or so to boot.
    while test (math (date +%s) - $start) -lt 180
        if _dock_up $sock
            echo "dock: $title is up after "(math (date +%s) - $start)'s' >&2
            return
        end
        sleep 1
    end
    echo "dock: $title didn't answer on $sock within 3 minutes" >&2
    return 1
end

# One row per engine, and one for docker's own context: whether it runs, and
# which one this shell and new shells use.
function _dock_list
    for engine in rancher docker podman
        set -l sock (_dock_socket $engine)
        set -l state stopped
        set -l marks
        if test -z "$sock"
            set state 'not set up'
        else
            _dock_up $sock; and set state running
            test "$DOCKER_HOST" = unix://$sock; and set -a marks 'this shell'
            test "$dock_default" = unix://$sock; and set -a marks default
        end
        printf '%-8s %-16s %-14s %s\n' $engine (_dock_title $engine) $state (string join ', ' $marks)
    end

    set -l config ~/.docker
    set -q DOCKER_CONFIG; and set config $DOCKER_CONFIG
    set -l context
    test -r $config/config.json
    and string match -rq '"currentContext":\s*"(?<context>[^"]+)"' <$config/config.json
    or set context default
    set -l marks
    set -q DOCKER_HOST; or set -a marks 'this shell'
    set -q dock_default; or set -a marks default
    printf '%-8s %-16s %-14s %s\n' off 'Docker context' $context (string join ', ' $marks)
end
