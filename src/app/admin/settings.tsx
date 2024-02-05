"use client";

import React from "react";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { QuestionMarkCircledIcon, GearIcon } from "@radix-ui/react-icons";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { translation } from "@/translation";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useForm } from "react-hook-form";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
} from "@/components/ui/form";
import { useMutation } from "react-query";
import axios from "axios";
import { useToast } from "@/components/ui/use-toast";

interface SettingsProps {
  initialConfig: SpotifyControllerConfig;
}

const formSchema = z.object({
  language: z.enum(["english", "korean"]),
  allowViewing: z.boolean(),
  allowSkipping: z.boolean(),
  allowPausing: z.boolean(),
});

export default function Settings(props: SettingsProps) {
  const { toast } = useToast();

  const [settings, setSettings] = React.useState<SpotifyControllerConfig>(
    props.initialConfig,
  );

  const t = translation[settings.language || "korean"].adminSettings;

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: props.initialConfig,
  });

  const { mutate } = useMutation(
    "updateConfig",
    async (data: SpotifyControllerConfig) => {
      await axios.post("/api/config", data);
      toast({ description: t.toastMessage });
    },
  );

  return (
    <main className="container grid min-h-screen place-items-center">
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit((data) => {
            mutate(data);
            setSettings(data);
          })}
        >
          <Card className="min-w-96">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <GearIcon />
                {t.title}
              </CardTitle>
              <CardDescription>{t.description}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4">
              <FormField
                name="allowViewing"
                control={form.control}
                render={({ field }) => (
                  <FormItem className="flex items-center space-x-2">
                    <FormControl>
                      <Switch
                        id="allowViewing"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <FormLabel htmlFor="allowViewing">
                      {t.allowViewingSwitch}
                    </FormLabel>
                    <FormDescription>
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <QuestionMarkCircledIcon className="cursor-pointer opacity-75" />
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>{t.allowViewingDescription}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </FormDescription>
                  </FormItem>
                )}
              />
              <FormField
                name="allowPausing"
                control={form.control}
                render={({ field }) => (
                  <FormItem className="flex items-center space-x-2">
                    <FormControl>
                      <Switch
                        id="allowPausing"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <FormLabel htmlFor="allowPausing">
                      {t.allowPausingSwitch}
                    </FormLabel>
                    <FormDescription>
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <QuestionMarkCircledIcon className="cursor-pointer opacity-75" />
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>{t.allowPausingDescription}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </FormDescription>
                  </FormItem>
                )}
              />
              <FormField
                name="allowSkipping"
                control={form.control}
                render={({ field }) => (
                  <FormItem className="flex items-center space-x-2">
                    <FormControl>
                      <Switch
                        id="allowSkipping"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <FormLabel htmlFor="allowSkipping">
                      {t.allowSkippingSwitch}
                    </FormLabel>
                    <FormDescription>
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <QuestionMarkCircledIcon className="cursor-pointer opacity-75" />
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>{t.allowSkippingDescription}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </FormDescription>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="language"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Language</FormLabel>
                    <Select
                      onValueChange={field.onChange}
                      defaultValue={field.value}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Language" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="english">English</SelectItem>
                        <SelectItem value="korean">한국어</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormItem>
                )}
              />
            </CardContent>
            <CardFooter className="grid space-y-2">
              <Button className="w-full" type="submit">
                {t.saveButton}
              </Button>
              <Button variant="destructive" className="w-full" type="button">
                {t.exitButton}
              </Button>
            </CardFooter>
          </Card>
        </form>
      </Form>
    </main>
  );
}
