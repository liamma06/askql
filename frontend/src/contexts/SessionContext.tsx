"use client";

import { createContext, useContext, useState, ReactNode } from "react";

interface SessionContextType {
  sessionActive: boolean;
  sessionId: string | null;
  //this sort of function takes in () and returns nothing
  setSessionActive: (active: boolean) => void;
  setSessionId: (id: string | null) => void;
  endSession: () => void;
}

const SessionContext = createContext<SessionContextType | undefined>(undefined);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [sessionActive, setSessionActive] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);

  const endSession = async () => {
    if (sessionId) {
      try {
        const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
        const response = await fetch(`${backendUrl}/api/session/${sessionId}`, {
          method: 'DELETE',
        });
        
        if (response.ok) {
          console.log('Session deleted successfully');
        } else {
          console.error('Failed to delete session:', response.status);
        }
      } catch (error) {
        console.error('Error deleting session:', error);
      }
    }
    
    // Reset local state regardless of API call result
    setSessionActive(false);
    setSessionId(null);
  };

  return (
    <SessionContext.Provider value={{ sessionActive, sessionId, setSessionActive, setSessionId, endSession }}>
      {children}
    </SessionContext.Provider>
  );
}

//global hook for session
export function useSession() {
  const context = useContext(SessionContext);
  if (context === undefined) {
    throw new Error("useSession must be used within a SessionProvider");
  }
  return context;
}
