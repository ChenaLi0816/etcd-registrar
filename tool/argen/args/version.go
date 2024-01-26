package args

const versionInfo = "v0.0.3"
const helpInfo = `-I --proto_path optional, proto file include path
-g --grpc optional, grpc grade generator
-i --implement optional, implement grade generator
-r --register optional, register grade generator
-v --version version info
-h --help help info

Last args are required protofiles, you can enter multiple files split with space.
With flag -v or -h existing, other flag will lapse.`
