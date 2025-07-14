import React, { useState, useEffect, useCallback, useRef } from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';

// 1. 类型定义
type User = {
    id: number;
    name: string;
    age: number;
};

// 2. 函数组件
const HomePage: React.FC = () => {
    return (
        <div className="home">
            <h1>Welcome to React App</h1>
            <Navigation />
        </div>
    );
};

// 3. 状态管理
const Counter: React.FC = () => {
    const [count, setCount] = useState(0);
    const [isLoading, setIsLoading] = useState(false);
    const ref = useRef<HTMLButtonElement>(null);

    useEffect(() => {
        document.title = `Count: ${count}`;
        return () => {
            // 清理副作用
        };
    }, [count]);

    const increment = useCallback(() => {
        setCount(prev => prev + 1);
    }, []);

    const fetchData = async () => {
        setIsLoading(true);
        // 模拟API调用
        setTimeout(() => {
            setIsLoading(false);
        }, 1000);
    };

    return (
        <div className="counter">
            <h2>Counter: {count}</h2>
            <button ref={ref} onClick={increment}>+1</button>
            <button onClick={fetchData} disabled={isLoading}>
                {isLoading ? 'Loading...' : 'Fetch Data'}
            </button>
        </div>
    );
};

// 4. 列表渲染
const UserList: React.FC<{ users: User[] }> = ({ users }) => {
    return (
        <ul className="user-list">
            {users.map(user => (
                <UserItem key={user.id} user={user} />
            ))}
        </ul>
    );
};

// 5. 子组件
const UserItem: React.FC<{ user: User }> = ({ user }) => {
    return (
        <li className="user-item">
            <span>{user.name}, {user.age}</span>
            <button onClick={() => console.log(`Selected ${user.name}`)}>
                Select
            </button>
        </li>
    );
};

// 6. 条件渲染
const LoadingIndicator: React.FC<{ isLoading: boolean }> = ({ isLoading }) => {
    if (!isLoading) return null;
    return <div className="loader">Loading...</div>;
};

// 7. 上下文API
const UserContext = React.createContext<User | null>(null);

const Profile: React.FC = () => {
    const user = React.useContext(UserContext);
    return (
        <div className="profile">
            {user ? (
                <div>
                    <h2>Profile: {user.name}</h2>
                    <p>Age: {user.age}</p>
                </div>
            ) : (
                <p>No user data</p>
            )}
        </div>
    );
};

// 8. 路由
const Navigation: React.FC = () => {
    return (
        <nav className="navigation">
            <Link to="/">Home</Link>
            <Link to="/counter">Counter</Link>
            <Link to="/users">Users</Link>
            <Link to="/profile">Profile</Link>
        </nav>
    );
};

// 9. 错误边界
class ErrorBoundary extends React.Component<
    React.PropsWithChildren,
    { hasError: boolean }
> {
    state = { hasError: false };

    static getDerivedStateFromError() {
        return { hasError: true };
    }

    componentDidCatch(error: Error) {
        console.error('ErrorBoundary caught an error:', error);
    }

    render() {
        if (this.state.hasError) {
            return <div className="error">Something went wrong.</div>;
        }
        return this.props.children;
    }
}

// 10. 自定义hook
const useWindowSize = () => {
    const [size, setSize] = useState({ width: 0, height: 0 });

    useEffect(() => {
        const handleResize = () => {
            setSize({ width: window.innerWidth, height: window.innerHeight });
        };

        window.addEventListener('resize', handleResize);
        handleResize(); // 初始调用

        return () => window.removeEventListener('resize', handleResize);
    }, []);

    return size;
};

// 11. 主应用
const App: React.FC = () => {
    const [users, setUsers] = useState<User[]>([]);
    const size = useWindowSize();

    useEffect(() => {
        // 模拟数据获取
        setUsers([
            { id: 1, name: 'Alice', age: 30 },
            { id: 2, name: 'Bob', age: 25 },
            { id: 3, name: 'Charlie', age: 35 },
        ]);
    }, []);

    return (
        <Router>
            <div className="app">
                <h1>React App</h1>
                <p>Window size: {size.width} x {size.height}</p>

                <ErrorBoundary>
                    <Routes>
                        <Route path="/" element={<HomePage />} />
                        <Route path="/counter" element={<Counter />} />
                        <Route
                            path="/users"
                            element={<UserList users={users} />}
                        />
                        <Route
                            path="/profile"
                            element={
                                <UserContext.Provider value={users[0]}>
                                    <Profile />
                                </UserContext.Provider>
                            }
                        />
                    </Routes>
                </ErrorBoundary>

                <LoadingIndicator isLoading={false} />
            </div>
        </Router>
    );
};

export default App;