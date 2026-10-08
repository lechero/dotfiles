function _tide_item_dock --description 'tide prompt item: the container engine `dock` pointed this shell at'
    # Each engine has its own colors in conf.d/tide.fish, as tide_dock_<engine>_*.
    switch "$DOCKER_HOST"
        case ''
            return
        case '*/.rd/docker.sock'
            _tide_print_item dock_rancher $tide_dock_icon' ' rancher
        case '*/.docker/run/docker.sock'
            _tide_print_item dock_docker $tide_dock_icon' ' docker
        case '*/podman/*'
            _tide_print_item dock_podman $tide_dock_icon' ' podman
        case '*'
            _tide_print_item dock $tide_dock_icon' ' (string replace -r '^unix://' '' -- $DOCKER_HOST)
    end
end
