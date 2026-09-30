import { useNavigate } from 'react-router-dom'
import { selfApi } from '@/lib/client'
import { setToken } from '@/lib/utils'
import { switchInstance, SELF } from '@/lib/instance'
import { LoginForm } from 'cheval-ui'

export default function Login() {
  const navigate = useNavigate()
  const handleSubmit = async (username: string, password: string) => {
    try {
      const { token } = await selfApi.apiAuthLoginPost({ TypesLoginRequest: { username, password } })
      setToken(token)
      switchInstance(SELF)
      navigate('/')
    } catch (err: unknown) {
      const msg = (err as Error).message
      throw Object.assign(
        new Error(msg === 'Unauthorized' ? 'Invalid username or password' : (msg || 'Login failed. Please check your credentials.')),
        { cause: err },
      )
    }
  }

  return <LoginForm description="Enter your credentials to access Routier" onSubmit={handleSubmit} />
}
