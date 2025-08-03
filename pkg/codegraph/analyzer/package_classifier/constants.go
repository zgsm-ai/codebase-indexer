package packageclassifier

// PackageType 定义包的类型
type PackageType string

const (
	SystemPackage     PackageType = "system"      // 系统包
	ThirdPartyPackage PackageType = "third_party" // 第三方包
	ProjectPackage    PackageType = "project"     // 项目内包
	UnknownPackage    PackageType = "unknown"     // 未知包
)
