# Script to merge generated mocks.


mock_pattern=mock_*.go
mock_merge_file=mocks.go

function update_mocks() {
    path=${1:-.}
    find ${path} -name "${mock_merge_file}" -delete

    echo "merge mocks: ${path}"
    $(merge_mocks $path)

    echo "remove generated mocks by pattern: ${mock_pattern}"
    $(rm_mocks $path)
}

function merge_mocks() {
  path=${1:-.}
  patter=${path}/${mock_pattern}
  result_file=${path}/${mock_merge_file}

  first=true
  for f in ${patter}; do
      if [ "${first}" = true ] ; then
        (cat "${f}"; echo; echo;)  >> ${result_file}
        first=false
      else
        import_end_line=$(awk -v line=')' '$0 == line {print NR}'  $f)
        file_lines=$(wc -l < ${f})
        tail_lines=$((${file_lines} - ${import_end_line}))
        (cat "${f}" | tail -n ${tail_lines}; echo; echo;) >> ${result_file}
      fi
  done

  exit 0
}

function rm_mocks() {
    path=${1:-.}
    find ${path} -name "${mock_pattern}" -delete

    exit 0
}

"$@"
