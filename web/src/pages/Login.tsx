import { selfApi } from '@/lib/client'
import { SELF, switchInstance } from '@/lib/instance'
import { setToken } from '@/lib/utils'
import { LoginForm } from 'cheval-ui'
import { useNavigate } from 'react-router-dom'

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
        new Error(msg === 'Unauthorized' ? 'Invalid username or password' : (msg || 'Sign in failed, check the username and password')),
        { cause: err },
      )
    }
  }

  return <LoginForm onSubmit={handleSubmit} />
}
