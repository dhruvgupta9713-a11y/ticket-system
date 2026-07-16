const API_BASE_URL = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
  ? 'http://localhost:8080'
  : '';

// Helper to make API requests with standard headers and error handling
async function request(endpoint, options = {}) {
  const url = `${API_BASE_URL}${endpoint}`;
  
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  const config = {
    ...options,
    headers,
  };

  if (options.body) {
    config.body = JSON.stringify(options.body);
  }

  const response = await fetch(url, config);
  const data = await response.json().catch(() => ({}));

  if (!response.ok) {
    const errorMsg = data.error || `HTTP error! status: ${response.status}`;
    throw new Error(errorMsg);
  }

  return data;
}

export const api = {
  // Registers a new user
  register: (username, email, password) => {
    return request('/auth/register', {
      method: 'POST',
      body: { username, email, password },
    });
  },

  // Log in user and return token
  login: (usernameOrEmail, password) => {
    // Determine whether they typed an email or a username
    const payload = usernameOrEmail.includes('@')
      ? { email: usernameOrEmail, password }
      : { username: usernameOrEmail, password };

    return request('/auth/login', {
      method: 'POST',
      body: payload,
    });
  },

  // Create a new ticket (Auth required)
  createTicket: (title, description, token) => {
    return request('/tickets', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: { title, description },
    });
  },

  // List all tickets for the logged-in user (Auth required)
  listTickets: (token) => {
    return request('/tickets', {
      method: 'GET',
      headers: { Authorization: `Bearer ${token}` },
    });
  },

  // Get a single ticket by ID (Auth required)
  getTicketByID: (id, token) => {
    return request(`/tickets/${id}`, {
      method: 'GET',
      headers: { Authorization: `Bearer ${token}` },
    });
  },

  // Update a ticket's status (Auth required)
  updateTicketStatus: (id, status, token) => {
    return request(`/tickets/${id}/status`, {
      method: 'PATCH',
      headers: { Authorization: `Bearer ${token}` },
      body: { status },
    });
  },
};
