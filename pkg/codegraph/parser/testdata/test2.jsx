import React, { useState } from 'react';

function App() {
  const [count, setCount] = useState(0);

  const handleClick = () => {
    setCount(count + 1);
  };

  return (
    <div style={styles.container}>
      <h1>Hello, React 👋</h1>
      <p>You clicked <strong>{count}</strong> times.</p>
      <button onClick={handleClick}>Click me</button>
    </div>
  );
}

const styles = {
  container: {
    textAlign: 'center',
    marginTop: '50px',
    fontFamily: 'sans-serif'
  }
};

export default App;