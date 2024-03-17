go_library_name = 'libp2p_wrapper.so'

go_library = custom_target('go_library',
  input: 'build.py',
  output: [go_library_name, 'libp2p_wrapper.h'],
  command: [python, '@INPUT@', '@SOURCE_ROOT@', '@OUTPUT0@'],
)

includes += include_directories('.')
go_library_dep = declare_dependency (sources: [go_library[0], go_library[1]])
deps += go_library_dep

lib_ldflags += join_paths(meson.current_source_dir(), go_library_name)