export default function Home() {
  return (
    <main style={{ padding: '2rem', fontFamily: 'sans-serif' }}>
      <h1>EnvNexus Console</h1>
      <p>Welcome to the EnvNexus Open Platform MVP.</p>
      
      <div style={{ marginTop: '2rem' }}>
        <h2>Quick Links</h2>
        <ul>
          <li><a href="/login">Login</a></li>
          <li><a href="/overview">Dashboard Overview</a></li>
        </ul>
      </div>
    </main>
  )
}
