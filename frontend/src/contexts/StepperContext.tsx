import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'

const STEPS_COUNT = 8
const STORAGE_KEY = 'stepper_state'

interface StepperState {
  maxReachedStep: number
  completedSteps: boolean[]
}

interface StepperContextType extends StepperState {
  markComplete: (step: number) => void
  canNavigateTo: (step: number) => boolean
  resetProgress: (toStep: number) => void
}

const StepperContext = createContext<StepperContextType | undefined>(undefined)

function loadState(): StepperState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      return {
        maxReachedStep: parsed.maxReachedStep ?? 0,
        completedSteps: parsed.completedSteps ?? Array(STEPS_COUNT).fill(false),
      }
    }
  } catch {
    // ignore
  }
  return {
    maxReachedStep: 0,
    completedSteps: Array(STEPS_COUNT).fill(false),
  }
}

function saveState(state: StepperState) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
}

export function StepperProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<StepperState>(loadState)

  const markComplete = useCallback((step: number) => {
    setState((prev) => {
      const completedSteps = [...prev.completedSteps]
      completedSteps[step] = true
      const maxReachedStep = Math.max(prev.maxReachedStep, step + 1)
      const next = { maxReachedStep, completedSteps }
      saveState(next)
      return next
    })
  }, [])

  const canNavigateTo = useCallback((step: number) => {
    return step <= state.maxReachedStep
  }, [state.maxReachedStep])

  const resetProgress = useCallback((toStep: number) => {
    const completedSteps = Array(STEPS_COUNT).fill(false)
    for (let i = 0; i < toStep; i++) completedSteps[i] = true
    const next = { maxReachedStep: toStep, completedSteps }
    saveState(next)
    setState(next)
  }, [])

  return (
    <StepperContext.Provider value={{ ...state, markComplete, canNavigateTo, resetProgress }}>
      {children}
    </StepperContext.Provider>
  )
}

export function useStepper(): StepperContextType {
  const context = useContext(StepperContext)
  if (!context) {
    throw new Error('useStepper must be used within a StepperProvider')
  }
  return context
}
