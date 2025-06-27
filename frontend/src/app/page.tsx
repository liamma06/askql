"use client";

import { useState,useEffect} from "react";
import Image from "next/image";
import { useSession } from "@/contexts/SessionContext";
import { useRouter } from "next/navigation";


export default function Home() {
  const router = useRouter();
  const { setSessionActive, setSessionId, sessionActive } = useSession();
  const [file, setFile] = useState<File | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadSuccess, setUploadSuccess] = useState(false);
  const [usingTestFile, setUsingTestFile] = useState(false);
  const [backendStatus, setBackendStatus] = useState<string>('checking...');
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);

  // Test backend connection on component mount
  useEffect(() => {
    const testBackendConnection = async () => {
      try {
        const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
        console.log('Testing backend connection to:', backendUrl);
        
        const response = await fetch(`${backendUrl}/health`);
        
        if (response.ok) {
          const data = await response.json();
          console.log('Backend health check response:', data);
          setBackendStatus('connected ✅');
        } else {
          console.error('Backend health check failed:', response.status);
          setBackendStatus('failed ❌');
        }
      } catch (error) {
        console.error('Backend connection error:', error);
        setBackendStatus('error ❌');
      }
    };

    testBackendConnection();
  }, []);

  // Reset upload success when session becomes inactive
  useEffect(() => {
    if (!sessionActive) {
      setUploadSuccess(false);
      setCurrentSessionId(null);
      setFile(null);
    }
  }, [sessionActive]);

  //function for new session
  const createSession = async (): Promise<string> => {
    const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    const response = await fetch(`${backendUrl}/api/session/create`, {
      method: 'POST',
    });
    if (!response.ok) throw new Error('Failed to create session');
    const data = await response.json();
    return data.session_id;
  };

  //success
  const handleUploadSuccess = (sessionId: string) => {
    setCurrentSessionId(sessionId);
    setSessionId(sessionId); // Update global session context
    setUploadSuccess(true);
    setSessionActive(true);
    setFile(null); // Clear the selected file
    console.log('Upload successful! Session ID:', sessionId);
    
    // Redirect to query page after successful upload
    router.push('/query');
  };


  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
      setUploadSuccess(false);
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const droppedFile = e.dataTransfer.files[0];
      if (droppedFile.type === "text/csv" || droppedFile.name.endsWith(".csv")) {
        setFile(droppedFile);
        setUploadSuccess(false);
      } else {
        alert("Please upload a CSV file");
      }
    }
  };

  const handleUseTestFile = async () => {
    setIsUploading(true);
    setUsingTestFile(true);
    
    try {
      // Fetch the test.csv file from the public directory
      const response = await fetch('/test.csv');
      if (!response.ok) {
        throw new Error('Failed to fetch test file');
      }
      
      const csvContent = await response.text();
      const testFile = new File([csvContent], 'test.csv', { type: 'text/csv' });
      
      // Set it as the selected file and trigger upload
      setFile(testFile);
      
      // Now upload it normally like any other file
      const sessionId = await createSession();
      
      const formData = new FormData();
      formData.append('file', testFile);
      formData.append('session_id', sessionId);

      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const uploadResponse = await fetch(`${backendUrl}/api/upload`, {
        method: 'POST',
        body: formData,
      });
      
      if (uploadResponse.ok) {
        handleUploadSuccess(sessionId);
      } else {
        const errorText = await uploadResponse.text();
        console.error('Upload failed:', uploadResponse.status, errorText);
        throw new Error(`Failed to use test file: ${uploadResponse.status}`);
      }
    } catch (error) {
      console.error('Error using test file:', error);
      alert('Failed to use test file. Please try again.');
    } finally {
      setIsUploading(false);
    }
  };
  
  const handleSubmit = async () => {
    if (!file) return;

    setIsUploading(true);
    setUsingTestFile(false);

    try {
      const sessionId = await createSession();
      
      const formData = new FormData();
      formData.append('file', file);
      formData.append('session_id', sessionId);

      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const response = await fetch(`${backendUrl}/api/upload`, {
        method: 'POST',
        body: formData,
      });

      if (response.ok) {
        handleUploadSuccess(sessionId);
      } else {
        const errorText = await response.text();
        console.error('Upload failed:', response.status, errorText);
        throw new Error(`Upload failed: ${response.status}`);
      }
    } catch (error) {
      console.error('Error uploading file:', error);
      alert('Failed to upload file. Please try again.');
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <main className="flex h-auto flex-col items-center p-8 md:p-12">
      <div className="mb-10">
        <h1 className="text-4xl font-bold text-center text-stone-700">Query Your Data</h1>
        <div className="text-center mt-2">
          <span className="text-sm text-gray-600">Backend status: {backendStatus}</span>
        </div>
      </div>

      <div className="w-full max-w-2xl mt-6">
        <div
          className={`border-2 border-dashed rounded-lg p-8 flex flex-col items-center justify-center transition-colors ${
            isDragging
              ? "border-blue-500 bg-blue-50"
              : uploadSuccess
                ? "border-green-500 bg-green-50"
                : "border-gray-300 bg-gray-50 hover:bg-gray-100"
          }`}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
        >
          <div className="mb-5">
            <svg
              className={`w-16 h-16 ${
                uploadSuccess
                  ? "text-green-500"
                  : "text-gray-400"
              }`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
          </div>

          {uploadSuccess ? (
            <div className="text-center">
              <div className="text-lg font-medium text-green-600 mb-2">Upload Successful!</div>
              <p className="text-sm text-green-500 mb-2">Your file has been uploaded successfully.</p>
              {currentSessionId && (
                <p className="text-xs text-gray-600 mb-4">Session ID: {currentSessionId}</p>
              )}
              <div className="text-sm text-gray-600 mt-4 p-4 bg-gray-50 rounded-md">
                <p className="mb-2">🎉 Your data is ready for querying!</p>
                <p className="text-xs">Use the "End Session" button in the header when you're done.</p>
              </div>
            </div>
          ) : (
            <>
              <div className="text-lg font-medium text-gray-700 mb-2">
                {file ? file.name : "Drop your CSV file here"}
              </div>
              <p className="text-sm text-gray-500 mb-4">
                {file ? `${(file.size / 1024).toFixed(2)} KB` : "or click to browse"}
              </p>

              <div className="flex flex-col sm:flex-row gap-4 w-full max-w-xs">
                <label className="flex-1">
                  <input
                    type="file"
                    className="hidden"
                    accept=".csv"
                    onChange={handleFileChange}
                  />
                  <div className="w-full px-4 py-2 text-center text-gray-600 bg-white border border-gray-300 rounded-md cursor-pointer hover:bg-gray-50 transition-colors">
                    Browse Files
                  </div>
                </label>

                {file && (
                  <button
                    className={`flex-1 px-4 py-2 text-white rounded-md transition-colors ${
                      isUploading
                        ? "bg-stone-500 cursor-not-allowed"
                        : "bg-stone-700 hover:bg-stone-800"
                    }`}
                    onClick={handleSubmit}
                    disabled={isUploading}
                  >
                    {isUploading ? "Uploading..." : "Upload"}
                  </button>
                )}
              </div>
              
              <div className="text-center mt-3">
                <div className="text-sm text-gray-500 mb-1">or</div>
                <button
                  onClick={handleUseTestFile}
                  disabled={isUploading}
                  className="text-blue-600 hover:text-blue-800 transition-colors font-medium underline disabled:opacity-50"
                >
                  {isUploading ? "Loading..." : "Use sample test.csv instead"}
                </button>
              </div>
            </>
          )}
        </div>

        <div className="mt-4 text-sm text-gray-500 text-center">
          Supported format: CSV files only
        </div>
      </div>
    </main>
  );
}


