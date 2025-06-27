"use client";

import { useState, useEffect } from "react";
import { useSession } from "@/contexts/SessionContext";
import { useRouter } from "next/navigation";

export default function QueryPage() {
  const router = useRouter();
  const { sessionActive, sessionId } = useSession();
  const [currentData, setCurrentData] = useState<any[]>([]);
  const [isLoadingData, setIsLoadingData] = useState(true);
  const [query, setQuery] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [results, setResults] = useState<any[]>([]);
  const [error, setError] = useState("");
  const [queryHistory, setQueryHistory] = useState<{query: string, timestamp: Date, type: 'sql' | 'natural'}[]>([]);
  const [isNaturalLanguage, setIsNaturalLanguage] = useState(false);
  const [showHistory, setShowHistory] = useState(false);

  // Redirect to home if no active session
  useEffect(() => {
    if (!sessionActive || !sessionId) {
      router.push('/');
    }
  }, [sessionActive, sessionId, router]);

  // Load current data when component mounts
  useEffect(() => {
    const loadCurrentData = async () => {
      if (!sessionId) return;

      setIsLoadingData(true);
      try {
        const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
        const response = await fetch(`${backendUrl}/api/query`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            sql: `SELECT * FROM data_${sessionId} LIMIT 100`,
            session_id: sessionId,
          }),
        });

        if (response.ok) {
          const data = await response.json();
          setCurrentData(data.data || []);
        } else {
          console.error('Failed to load current data');
        }
      } catch (error) {
        console.error('Error loading current data:', error);
      } finally {
        setIsLoadingData(false);
      }
    };

    if (sessionActive && sessionId) {
      loadCurrentData();
    }
  }, [sessionActive, sessionId]);

  const handleQuerySubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim() || !sessionId) return;

    setIsLoading(true);
    setError("");

    try {
      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      
      let processedQuery = query;
      let endpoint = '/api/query';
      
      if (isNaturalLanguage) {
        // Use the correct natural language endpoint that exists in backend
        endpoint = '/api/natural';
      } else {
        // Replace 'data' with the full table name for SQL queries
        processedQuery = query.replace(/\bdata\b/g, `data_${sessionId}`);
      }
      
      const response = await fetch(`${backendUrl}${endpoint}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          [isNaturalLanguage ? 'query' : 'sql']: isNaturalLanguage ? query : processedQuery,
          session_id: sessionId,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setResults(data.data || []);
        
        // Add to query history
        setQueryHistory(prev => [{
          query: query,
          timestamp: new Date(),
          type: isNaturalLanguage ? 'natural' : 'sql'
        }, ...prev.slice(0, 9)]); // Keep last 10 queries
        
      } else {
        const errorText = await response.text();
        setError(`Query failed: ${errorText}`);
      }
    } catch (error) {
      console.error('Error executing query:', error);
      setError('Failed to execute query. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  if (!sessionActive || !sessionId) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <p>Redirecting to upload page...</p>
        </div>
      </div>
    );
  }

  return (
    <main className="container mx-auto p-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-stone-700 mb-2">Query Your Data</h1>
        <p className="text-gray-600">Session ID: {sessionId}</p>
      </div>

      {/* Current Database Section */}
      <div className="mb-8">
        <h2 className="text-xl font-semibold text-gray-800 mb-4">Current Database</h2>
        <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
          <div className="px-6 py-4 bg-gray-50 border-b border-gray-200">
            <h3 className="text-lg font-medium text-gray-900">
              {isLoadingData ? "Loading..." : `Your Data (${currentData.length} rows)`}
            </h3>
          </div>
          <div className="max-h-96 overflow-auto">
            {isLoadingData ? (
              <div className="flex items-center justify-center p-8">
                <div className="text-gray-500">Loading your data...</div>
              </div>
            ) : currentData.length > 0 ? (
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50 sticky top-0">
                  <tr>
                    {Object.keys(currentData[0] || {}).map((key) => (
                      <th
                        key={key}
                        className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                      >
                        {key}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {currentData.map((row, index) => (
                    <tr key={index} className={index % 2 === 0 ? "bg-white" : "bg-gray-50"}>
                      {Object.values(row).map((value, cellIndex) => (
                        <td key={cellIndex} className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          {String(value)}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <div className="flex items-center justify-center p-8">
                <div className="text-gray-500">No data available</div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* SQL Query Section */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-gray-800">Query Interface</h2>
          <div className="flex items-center space-x-4">
            {/* Query History Toggle */}
            <button
              onClick={() => setShowHistory(!showHistory)}
              className="px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 transition-colors"
            >
              {showHistory ? 'Hide History' : 'Show History'} ({queryHistory.length})
            </button>
            
            {/* SQL/Natural Language Toggle */}
            <div className="flex items-center space-x-2">
              <span className={`text-sm ${!isNaturalLanguage ? 'text-stone-700 font-medium' : 'text-gray-500'}`}>
                SQL
              </span>
              <button
                onClick={() => setIsNaturalLanguage(!isNaturalLanguage)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  isNaturalLanguage ? 'bg-stone-600' : 'bg-gray-300'
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    isNaturalLanguage ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
              <span className={`text-sm ${isNaturalLanguage ? 'text-stone-700 font-medium' : 'text-gray-500'}`}>
                Natural Language
              </span>
            </div>
          </div>
        </div>

        {/* Query History Section */}
        {showHistory && queryHistory.length > 0 && (
          <div className="mb-6 bg-gray-50 border border-gray-200 rounded-lg overflow-hidden">
            <div className="px-4 py-3 bg-gray-100 border-b border-gray-200">
              <h3 className="text-sm font-medium text-gray-700">Query History</h3>
            </div>
            <div className="max-h-48 overflow-auto">
              {queryHistory.map((historyItem, index) => (
                <div
                  key={index}
                  className="px-4 py-3 border-b border-gray-200 last:border-b-0 hover:bg-gray-100 cursor-pointer transition-colors"
                  onClick={() => {
                    setQuery(historyItem.query);
                    setIsNaturalLanguage(historyItem.type === 'natural');
                  }}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className={`text-xs px-2 py-1 rounded ${
                      historyItem.type === 'natural' 
                        ? 'bg-blue-100 text-blue-700' 
                        : 'bg-green-100 text-green-700'
                    }`}>
                      {historyItem.type === 'natural' ? 'Natural Language' : 'SQL'}
                    </span>
                    <span className="text-xs text-gray-500">
                      {historyItem.timestamp.toLocaleTimeString()}
                    </span>
                  </div>
                  <p className="text-sm text-gray-700 truncate">{historyItem.query}</p>
                </div>
              ))}
            </div>
          </div>
        )}

        <form onSubmit={handleQuerySubmit} className="space-y-4">
          <div>
            <label htmlFor="query" className="block text-sm font-medium text-gray-700 mb-2">
              {isNaturalLanguage ? 'Natural Language Query' : 'SQL Query'}
            </label>
            
            {!isNaturalLanguage && (
              <div className="mb-2 p-3 bg-blue-50 border border-blue-200 rounded-md">
                <p className="text-sm text-blue-700">
                  💡 <strong>Tip:</strong> You can use <code className="bg-blue-100 px-1 rounded">data</code> as your table name instead of the full session ID.
                </p>
                <p className="text-xs text-blue-600 mt-1">
                  Example: <code className="bg-blue-100 px-1 rounded">SELECT * FROM data WHERE Name = 'John'</code>
                </p>
              </div>
            )}
            
            {isNaturalLanguage && (
              <div className="mb-2 p-3 bg-green-50 border border-green-200 rounded-md">
                <p className="text-sm text-green-700">
                  🗣️ <strong>Natural Language:</strong> Ask questions about your data in plain English.
                </p>
                <p className="text-xs text-green-600 mt-1">
                  Example: <code className="bg-green-100 px-1 rounded">"Show me all records where age is greater than 25"</code>
                </p>
              </div>
            )}
            
            <textarea
              id="query"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={isNaturalLanguage 
                ? "Ask a question about your data in plain English..."
                : "SELECT * FROM data WHERE..."
              }
              className="w-full p-3 border border-gray-300 rounded-md focus:ring-2 focus:ring-stone-500 focus:border-transparent"
              rows={4}
            />
          </div>
          <button
            type="submit"
            disabled={isLoading || !query.trim()}
            className="px-6 py-2 bg-stone-700 text-white rounded-md hover:bg-stone-800 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading 
              ? (isNaturalLanguage ? "Processing..." : "Executing...") 
              : (isNaturalLanguage ? "Ask Question" : "Execute Query")
            }
          </button>
        </form>
      </div>

      {/* Error Display */}
      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-md">
          <p className="text-red-600">{error}</p>
        </div>
      )}

      {/* Query Results Display */}
      {results.length > 0 && (
        <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
          <div className="px-6 py-4 bg-gray-50 border-b border-gray-200">
            <h3 className="text-lg font-medium text-gray-900">
              Query Results ({results.length} rows)
            </h3>
          </div>
          <div className="max-h-96 overflow-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50 sticky top-0">
                <tr>
                  {Object.keys(results[0] || {}).map((key) => (
                    <th
                      key={key}
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                    >
                      {key}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {results.map((row, index) => (
                  <tr key={index} className={index % 2 === 0 ? "bg-white" : "bg-gray-50"}>
                    {Object.values(row).map((value, cellIndex) => (
                      <td key={cellIndex} className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                        {String(value)}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </main>
  );
}
