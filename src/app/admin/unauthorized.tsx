import Link from "next/link";
import { ExclamationTriangleIcon } from "@radix-ui/react-icons";
import { Button } from "@/components/ui/button";
import { translation } from "@/translation";
import { getConfiguration } from "@/lib/config";

export default async function Unauthorized() {
  const config = await getConfiguration();
  const t = translation[config.language].adminLogin;

  return (
    <main className="container grid min-h-screen place-items-center">
      <section className="grid place-items-center space-y-2">
        <ExclamationTriangleIcon className="h-8 w-8" />
        <h1 className="text-xl uppercase">{t.unauthorized}</h1>
        <Link href="/">
          <Button variant="destructive">{t.gobackButton}</Button>
        </Link>
      </section>
    </main>
  );
}
