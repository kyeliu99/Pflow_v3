import { useState } from "react";
import {
  Badge,
  Box,
  Button,
  Flex,
  FormControl,
  FormLabel,
  Heading,
  Input,
  Stack,
  Text,
  Textarea,
  useToast,
} from "@chakra-ui/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createForm, listForms } from "../lib/api";

export default function FormLibrary() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["forms"],
    queryFn: () => listForms(),
  });

  const forms = data?.data ?? [];

  const createMutation = useMutation({
    mutationFn: createForm,
    onSuccess: () => {
      toast({ status: "success", title: "表单已创建" });
      queryClient.invalidateQueries({ queryKey: ["forms"] });
      queryClient.invalidateQueries({ queryKey: ["overview"] });
      setName("");
      setDescription("");
    },
    onError: () => {
      toast({ status: "error", title: "创建表单失败", description: "请稍后重试" });
    },
  });

  const handleCreate = () => {
    if (!name.trim()) {
      toast({ status: "warning", title: "请输入表单名称" });
      return;
    }

    createMutation.mutate({
      name: name.trim(),
      description: description.trim(),
      schema: {
        fields: [],
      },
    });
  };

  return (
    <Box borderWidth="1px" borderRadius="md" p={4} bg="white" shadow="sm">
      <Heading size="md" mb={4}>
        表单库
      </Heading>
      <Stack spacing={4} mb={6}>
        <FormControl>
          <FormLabel>表单名称</FormLabel>
          <Input
            placeholder="例如：入职信息收集"
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </FormControl>
        <FormControl>
          <FormLabel>描述</FormLabel>
          <Textarea
            placeholder="概述该表单的用途"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
          />
        </FormControl>
        <Button
          colorScheme="blue"
          alignSelf="flex-start"
          isLoading={createMutation.isPending}
          onClick={handleCreate}
        >
          新建表单
        </Button>
      </Stack>

      <Heading size="sm" mb={2}>
        已有表单
      </Heading>
      {isLoading && <Text color="gray.500">加载中...</Text>}
      {!isLoading && forms.length === 0 && <Text color="gray.500">暂无表单。</Text>}
      <Stack spacing={3} mt={3}>
        {forms.map((form) => (
          <Box key={form.id} borderWidth="1px" borderRadius="md" p={3}>
            <Flex justify="space-between" align="center">
              <Box>
                <Heading size="sm">{form.name}</Heading>
                <Text fontSize="sm" color="gray.500">
                  {form.description || "无描述"}
                </Text>
              </Box>
              <Badge colorScheme="blue">ID: {form.id.slice(0, 8)}</Badge>
            </Flex>
          </Box>
        ))}
      </Stack>
    </Box>
  );
}
