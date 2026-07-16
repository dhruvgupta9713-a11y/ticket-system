import React, { useState, useEffect } from 'react';
import { api } from './api';

function App() {
  const [token, setToken] = useState(localStorage.getItem('ticket_token') || '');
  const [username, setUsername] = useState(localStorage.getItem('ticket_username') || '');
  const [view, setView] = useState(token ? 'dashboard' : 'login');
  
  // Login form state
  const [loginIdent, setLoginIdent] = useState('');
  const [loginPassword, setLoginPassword] = useState('');
  const [loginError, setLoginError] = useState('');
  const [loginLoading, setLoginLoading] = useState(false);

  // Register form state
  const [regUsername, setRegUsername] = useState('');
  const [regEmail, setRegEmail] = useState('');
  const [regPassword, setRegPassword] = useState('');
  const [regError, setRegError] = useState('');
  const [regSuccess, setRegSuccess] = useState('');
  const [regLoading, setRegLoading] = useState(false);

  // Dashboard state
  const [tickets, setTickets] = useState([]);
  const [ticketsError, setTicketsError] = useState('');
  const [ticketsLoading, setTicketsLoading] = useState(false);

  // Create Ticket Modal State
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newTitle, setNewTitle] = useState('');
  const [newDescription, setNewDescription] = useState('');
  const [createError, setCreateError] = useState('');
  const [createLoading, setCreateLoading] = useState(false);

  // Sync token and username to local storage and fetch tickets if authenticated
  useEffect(() => {
    if (token) {
      localStorage.setItem('ticket_token', token);
      localStorage.setItem('ticket_username', username);
      setView('dashboard');
      fetchTickets();
    } else {
      localStorage.removeItem('ticket_token');
      localStorage.removeItem('ticket_username');
    }
  }, [token, username]);

  const handleLogout = () => {
    setToken('');
    setUsername('');
    setView('login');
    setTickets([]);
  };

  const fetchTickets = async () => {
    if (!token) return;
    setTicketsLoading(true);
    setTicketsError('');
    try {
      const data = await api.listTickets(token);
      // Ensure the returned value is always an array
      setTickets(Array.isArray(data) ? data : []);
    } catch (err) {
      setTicketsError(err.message);
    } finally {
      setTicketsLoading(false);
    }
  };

  const handleLoginSubmit = async (e) => {
    e.preventDefault();
    if (!loginIdent || !loginPassword) {
      setLoginError('All fields are required');
      return;
    }
    setLoginLoading(true);
    setLoginError('');
    try {
      const data = await api.login(loginIdent, loginPassword);
      
      // Split JWT token payload to decode and retrieve user info
      let extractedUsername = loginIdent;
      try {
        const payloadBase64 = data.token.split('.')[1];
        const decodedPayload = JSON.parse(atob(payloadBase64));
        extractedUsername = decodedPayload.username || loginIdent;
      } catch (err) {
        console.warn('Failed to parse JWT payload', err);
      }
      
      setUsername(extractedUsername);
      setToken(data.token);
    } catch (err) {
      setLoginError(err.message);
    } finally {
      setLoginLoading(false);
    }
  };

  const handleRegisterSubmit = async (e) => {
    e.preventDefault();
    if (!regUsername || !regEmail || !regPassword) {
      setRegError('All fields are required');
      return;
    }
    setRegLoading(true);
    setRegError('');
    setRegSuccess('');
    try {
      await api.register(regUsername, regEmail, regPassword);
      setRegSuccess('Registration successful! Redirecting to login...');
      setRegUsername('');
      setRegEmail('');
      setRegPassword('');
      setTimeout(() => {
        setView('login');
        setRegSuccess('');
      }, 1500);
    } catch (err) {
      setRegError(err.message);
    } finally {
      setRegLoading(false);
    }
  };

  const handleCreateTicketSubmit = async (e) => {
    e.preventDefault();
    if (!newTitle || !newDescription) {
      setCreateError('Title and description are required');
      return;
    }
    setCreateLoading(true);
    setCreateError('');
    try {
      await api.createTicket(newTitle, newDescription, token);
      setNewTitle('');
      setNewDescription('');
      setShowCreateModal(false);
      fetchTickets();
    } catch (err) {
      setCreateError(err.message);
    } finally {
      setCreateLoading(false);
    }
  };

  const handleStatusTransition = async (ticketId, nextStatus) => {
    try {
      await api.updateTicketStatus(ticketId, nextStatus, token);
      fetchTickets();
    } catch (err) {
      alert(`Failed to transition status: ${err.message}`);
    }
  };

  const formatDate = (dateStr) => {
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <>
      {token && (
        <nav className="navbar">
          <div className="nav-brand">TicketSystem</div>
          <div className="nav-user-area">
            <span className="nav-username">Hello, {username}</span>
            <button className="btn btn-logout" onClick={handleLogout}>Logout</button>
          </div>
        </nav>
      )}

      {view === 'login' && (
        <div className="auth-container">
          <div className="card">
            <h2 className="card-title">Welcome Back</h2>
            <p className="card-subtitle">Sign in to manage your tickets</p>
            {loginError && <div className="alert-error">{loginError}</div>}
            <form onSubmit={handleLoginSubmit}>
              <div className="form-group">
                <label className="form-label">Username or Email</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder="Enter your username or email"
                  value={loginIdent}
                  onChange={(e) => setLoginIdent(e.target.value)}
                  disabled={loginLoading}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Password</label>
                <input
                  type="password"
                  className="form-input"
                  placeholder="••••••••"
                  value={loginPassword}
                  onChange={(e) => setLoginPassword(e.target.value)}
                  disabled={loginLoading}
                />
              </div>
              <button type="submit" className="btn btn-primary" disabled={loginLoading}>
                {loginLoading ? 'Signing in...' : 'Sign In'}
              </button>
            </form>
            <div className="auth-switch">
              Don't have an account?
              <span className="auth-link" onClick={() => { setView('register'); setLoginError(''); }}>
                Sign Up
              </span>
            </div>
          </div>
        </div>
      )}

      {view === 'register' && (
        <div className="auth-container">
          <div className="card">
            <h2 className="card-title">Create Account</h2>
            <p className="card-subtitle">Register to start creating tickets</p>
            {regError && <div className="alert-error">{regError}</div>}
            {regSuccess && <div className="alert-success">{regSuccess}</div>}
            <form onSubmit={handleRegisterSubmit}>
              <div className="form-group">
                <label className="form-label">Username</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder="Choose a username"
                  value={regUsername}
                  onChange={(e) => setRegUsername(e.target.value)}
                  disabled={regLoading || regSuccess}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Email Address</label>
                <input
                  type="email"
                  className="form-input"
                  placeholder="name@example.com"
                  value={regEmail}
                  onChange={(e) => setRegEmail(e.target.value)}
                  disabled={regLoading || regSuccess}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Password</label>
                <input
                  type="password"
                  className="form-input"
                  placeholder="••••••••"
                  value={regPassword}
                  onChange={(e) => setRegPassword(e.target.value)}
                  disabled={regLoading || regSuccess}
                />
              </div>
              <button type="submit" className="btn btn-primary" disabled={regLoading || regSuccess}>
                {regLoading ? 'Registering...' : 'Register'}
              </button>
            </form>
            <div className="auth-switch">
              Already have an account?
              <span className="auth-link" onClick={() => { setView('login'); setRegError(''); }}>
                Sign In
              </span>
            </div>
          </div>
        </div>
      )}

      {view === 'dashboard' && (
        <div className="dashboard">
          <div className="dashboard-header">
            <div>
              <h1 className="dashboard-title">Your Tickets</h1>
              <p style={{ color: 'var(--text-secondary)', marginTop: '0.25rem' }}>View and update your created tickets</p>
            </div>
            <button className="btn btn-create" onClick={() => setShowCreateModal(true)}>
              + Create Ticket
            </button>
          </div>

          {ticketsError && <div className="alert-error">{ticketsError}</div>}

          {ticketsLoading ? (
            <div style={{ textAlign: 'center', padding: '4rem', color: 'var(--text-secondary)' }}>
              Loading tickets...
            </div>
          ) : (
            <div className="tickets-grid">
              {tickets.length === 0 ? (
                <div className="no-tickets">
                  <h3>No Tickets Found</h3>
                  <p style={{ marginTop: '0.5rem', color: 'var(--text-muted)' }}>Create your first ticket to get started!</p>
                </div>
              ) : (
                tickets.map((ticket) => (
                  <div key={ticket.id} className="ticket-card">
                    <div className="ticket-card-header">
                      <h3 className="ticket-title">{ticket.title}</h3>
                      <span className={`badge badge-${ticket.status}`}>
                        {ticket.status.replace('_', ' ')}
                      </span>
                    </div>
                    <p className="ticket-desc">{ticket.description}</p>
                    
                    <div className="ticket-meta">
                      <div className="ticket-meta-item">Ticket ID: #{ticket.id}</div>
                      <div className="ticket-meta-item">Created: {formatDate(ticket.created_at)}</div>
                      <div className="ticket-meta-item">Updated: {formatDate(ticket.updated_at)}</div>
                    </div>

                    <div className="ticket-actions">
                      {ticket.status === 'open' && (
                        <button
                          className="btn btn-primary btn-action-status"
                          onClick={() => handleStatusTransition(ticket.id, 'in_progress')}
                        >
                          Start Progress
                        </button>
                      )}
                      {ticket.status === 'in_progress' && (
                        <button
                          className="btn btn-primary btn-action-status"
                          onClick={() => handleStatusTransition(ticket.id, 'closed')}
                          style={{ background: 'linear-gradient(135deg, #10B981 0%, #059669 100%)' }}
                        >
                          Close Ticket
                        </button>
                      )}
                      {ticket.status === 'closed' && (
                        <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem', fontStyle: 'italic' }}>
                          Ticket Resolved
                        </span>
                      )}
                    </div>
                  </div>
                ))
              )}
            </div>
          )}

          {showCreateModal && (
            <div className="modal-backdrop">
              <div className="modal-content">
                <h2 style={{ marginBottom: '0.5rem' }}>Create New Ticket</h2>
                <p style={{ color: 'var(--text-secondary)', marginBottom: '1.5rem', fontSize: '0.9rem' }}>Fill in details to open a new ticket</p>
                {createError && <div className="alert-error">{createError}</div>}
                <form onSubmit={handleCreateTicketSubmit}>
                  <div className="form-group">
                    <label className="form-label">Ticket Title</label>
                    <input
                      type="text"
                      className="form-input"
                      placeholder="e.g. Broken links in footer"
                      value={newTitle}
                      onChange={(e) => setNewTitle(e.target.value)}
                      disabled={createLoading}
                    />
                  </div>
                  <div className="form-group">
                    <label className="form-label">Description</label>
                    <textarea
                      className="form-input"
                      rows="4"
                      placeholder="Provide details about the issue..."
                      value={newDescription}
                      onChange={(e) => setNewDescription(e.target.value)}
                      style={{ resize: 'vertical' }}
                      disabled={createLoading}
                    />
                  </div>
                  <div className="modal-actions">
                    <button
                      type="button"
                      className="btn btn-cancel"
                      onClick={() => { setShowCreateModal(false); setCreateError(''); }}
                      disabled={createLoading}
                    >
                      Cancel
                    </button>
                    <button type="submit" className="btn btn-primary" disabled={createLoading}>
                      {createLoading ? 'Creating...' : 'Create'}
                    </button>
                  </div>
                </form>
              </div>
            </div>
          )}
        </div>
      )}
    </>
  );
}

export default App;
