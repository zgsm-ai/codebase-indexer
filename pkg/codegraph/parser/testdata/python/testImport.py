# 基本导入
import module
import module1, module2
import package.module
import package.subpackage.module

# 带别名导入
import module as alias
import module1 as alias1, module2 as alias2
import package.module as alias

# 基本from导入
from module import name
from module import name1, name2
from package.module import name
from package.subpackage.module import name

# 带别名from导入
from module import name as alias
from module import name1 as alias1, name2 as alias2
from module import name3 as alias3, name4 ,name5
# 导入所有
from module import *

from collections import (
    defaultdict,
    OrderedDict,
    Counter
)
from ..module11 import name
from ..package12 import module
from ..package.module13 import name as name1