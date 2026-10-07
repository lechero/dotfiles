# Fisher installs plugins into plugins/ instead of mixing them into
# functions/, conf.d/ and completions/, so those hold only this repo's files.
# The plugin list is fish_plugins; chezmoi runs `fisher update` whenever it
# changes.
set -g fisher_path $__fish_config_dir/plugins

if not contains -- $fisher_path/functions $fish_function_path
    set fish_function_path $fish_function_path[1] $fisher_path/functions $fish_function_path[2..]
    set fish_complete_path $fish_complete_path[1] $fisher_path/completions $fish_complete_path[2..]
end

# sdkman-for-fish warns at every start when SDKMAN isn't installed where it
# looks, as on a new machine, so it only loads where SDKMAN is. It looks in
# __sdkman_custom_dir, then SDKMAN_DIR, then ~/.sdkman; either variable may
# be left over from an older setup, inherited or universal.
set -l sdkman ~/.sdkman
set -q SDKMAN_DIR; and set sdkman $SDKMAN_DIR
set -q __sdkman_custom_dir; and set sdkman $__sdkman_custom_dir

for file in $fisher_path/conf.d/*.fish
    if test (path basename -- $file) = sdk.fish; and not test -f $sdkman/bin/sdkman-init.sh
        continue
    end
    source $file
end
