function _tide_item_kubectl --description 'tide prompt item: kubectl context and namespace, but not for local clusters'
    # Replaces tide's own kubectl item, as functions/ comes before plugins/ in
    # fish_function_path. Like tide_docker_default_contexts for docker, this
    # hides the contexts in tide_kubectl_default_contexts, the clusters Rancher
    # Desktop and Docker Desktop run, so the item only shows a real cluster.
    kubectl config view --minify --output 'jsonpath={.current-context}/{..namespace}' 2>/dev/null | read -l context &&
        not contains -- (string replace -r '/[^/]*$' '' -- $context) $tide_kubectl_default_contexts &&
        _tide_print_item kubectl $tide_kubectl_icon' ' (string replace -r '/(|default)$' '' $context)
end
