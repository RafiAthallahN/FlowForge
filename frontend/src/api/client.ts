import axios from 'axios'

const isDev = typeof window !== 'undefined' && window.location.port === '5173'
const baseURL = isDev ? 'http://localhost:8080/api' : '/api'

const client = axios.create({
  baseURL,
  headers: {
    'Content-Type': 'application/json',
  },
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('ff_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('ff_token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default client
