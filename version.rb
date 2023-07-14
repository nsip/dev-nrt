original_directory = Dir::pwd
nias_version = "N/A"
splitter_version = "N/A"
begin
Dir::chdir("../dev-nrt-splitter") do
  splitter2version = `git describe --tags`
end
Dir::chdir("..") do
  nias2version = `git describe --tags`
end
rescue
end
Dir::chdir(original_directory)

devnrt_version = `git describe --tags`
devnrt_branch = `git branch|grep '*'`
devnrt_commit = `git log|head -5`


print <<~END
package main

import (
        "fmt"
)

func showVersion() {
        fmt.Println("nias2 Version: #{nias_version.strip}")
        fmt.Println("dev-nrt-splitter Version: #{splitter_version.strip}")
        fmt.Println("dev-nrt Version: #{devnrt_version.strip}")
        fmt.Println("dev-nrt code branch: #{devnrt_branch.strip}")
        fmt.Println("dev-nrt last commit:\\n\\n#{devnrt_commit.strip.gsub(/\n/, "\\n")}")
}
END


