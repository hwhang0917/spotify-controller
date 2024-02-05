import { Toaster } from "@/components/ui/toaster";

export default function AdminLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <>
      {children}
      <Toaster />
    </>
  );
}
