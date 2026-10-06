# Fisher installs plugins into plugins/ instead of mixing them into
# functions/, conf.d/ and completions/, so those hold only this repo's files.
# The plugin list is fish_plugins; chezmoi runs `fisher update` whenever it
# changes.
set -g fisher_path $__fish_config_dir/plugins

if not contains -- $fisher_path/functions $fish_function_path
    set fish_function_path $fish_function_path[1] $fisher_path/functions $fish_function_path[2..]
    set fish_complete_path $fish_complete_path[1] $fisher_path/completions $fish_complete_path[2..]
end

for file in $fisher_path/conf.d/*.fish
    source $file
end
