function move-last-download --argument-names destination --description 'Move the newest file in ~/Downloads here, or to the given directory'
    set -l newest (command ls -t -A ~/Downloads)[1]
    if test -z "$newest"
        echo 'move-last-download: ~/Downloads is empty' >&2
        return 1
    end
    test -n "$destination"; or set destination .
    mv ~/Downloads/$newest $destination
end
