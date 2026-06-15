import { reactive } from 'vue'

type ToastTone = 'success' | 'error'

interface ToastState {
  visible: boolean
  message: string
  tone: ToastTone
}

export const toastState = reactive<ToastState>({
  visible: false,
  message: '',
  tone: 'success',
})

let timer: ReturnType<typeof setTimeout> | undefined

export function showToast(message: string, tone: ToastTone = 'success') {
  clearTimeout(timer)
  Object.assign(toastState, { visible: true, message, tone })
  timer = setTimeout(() => {
    toastState.visible = false
  }, 3500)
}

