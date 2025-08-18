DNSResolver::TaskHandle NativeDNSResolver::LookupHostname(
  std::function<void(absl::statusOr<std::vector<grpc resolved address>>)>
  on resolved,
  absl::string view name, absl::string view default port,
  Duration /* timeout */,grpc pollset set* /* interested parties */,
  absl::string view /* name server */){}