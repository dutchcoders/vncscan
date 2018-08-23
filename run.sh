while read p; do
  go run main.go $p
done < /dev/stdin
