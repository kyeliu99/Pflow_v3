import { useMemo, useState } from "react";
import {
  Badge,
  Box,
  Button,
  Flex,
  FormControl,
  FormLabel,
  Heading,
  HStack,
  IconButton,
  Input,
  Select,
  Stack,
  Text,
  Textarea,
  useToast,
} from "@chakra-ui/react";
import { CheckIcon, RepeatIcon } from "@chakra-ui/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createWorkflow,
  listWorkflows,
  publishWorkflow,
  WorkflowDefinition,
} from "../lib/api";

const steps = [
  { id: "form", label: "表单填写" },
  { id: "approval", label: "人工审批" },
  { id: "automation", label: "自动化执行" },
  { id: "notification", label: "通知提醒" },
];

export default function WorkflowDesigner() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [workflowName, setWorkflowName] = useState("");
  const [workflowDescription, setWorkflowDescription] = useState("");
  const [workflowSteps, setWorkflowSteps] = useState<string[]>([]);

  const { data: workflowsData, isLoading } = useQuery({
    queryKey: ["workflows"],
    queryFn: () => listWorkflows(),
  });

  const createMutation = useMutation({
    mutationFn: createWorkflow,
    onSuccess: () => {
      toast({ status: "success", title: "流程已创建" });
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      queryClient.invalidateQueries({ queryKey: ["overview"] });
      setWorkflowName("");
      setWorkflowDescription("");
      setWorkflowSteps([]);
    },
    onError: () => {
      toast({ status: "error", title: "创建流程失败", description: "请检查表单内容或稍后重试" });
    },
  });

  const publishMutation = useMutation({
    mutationFn: publishWorkflow,
    onSuccess: () => {
      toast({ status: "success", title: "流程已发布" });
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      queryClient.invalidateQueries({ queryKey: ["overview"] });
    },
    onError: () => {
      toast({ status: "error", title: "发布流程失败" });
    },
  });

  const workflows = workflowsData?.data ?? [];

  const blueprint = useMemo(() => {
    return {
      steps: workflowSteps.map((step, index) => ({
        id: `${index + 1}`,
        type: step,
        name: steps.find((item) => item.id === step)?.label ?? step,
      })),
    };
  }, [workflowSteps]);

  const handleCreateWorkflow = () => {
    if (!workflowName.trim()) {
      toast({ status: "warning", title: "请输入流程名称" });
      return;
    }
    if (workflowSteps.length === 0) {
      toast({ status: "warning", title: "请至少添加一个步骤" });
      return;
    }

    createMutation.mutate({
      name: workflowName.trim(),
      description: workflowDescription.trim(),
      blueprint,
    });
  };

  return (
    <Box borderWidth="1px" borderRadius="md" p={4} bg="white" shadow="sm">
      <Flex justify="space-between" align="center" mb={4}>
        <Heading size="md">流程编排器</Heading>
        <Button
          size="sm"
          leftIcon={<RepeatIcon />}
          onClick={() => {
            setWorkflowName("");
            setWorkflowDescription("");
            setWorkflowSteps([]);
          }}
        >
          重置
        </Button>
      </Flex>
      <Stack spacing={4} mb={6}>
        <FormControl>
          <FormLabel>流程名称</FormLabel>
          <Input
            placeholder="例如：入职审批流程"
            value={workflowName}
            onChange={(event) => setWorkflowName(event.target.value)}
          />
        </FormControl>
        <FormControl>
          <FormLabel>流程描述</FormLabel>
          <Textarea
            placeholder="描述流程的触发条件、目标、注意事项等"
            value={workflowDescription}
            onChange={(event) => setWorkflowDescription(event.target.value)}
          />
        </FormControl>
        <FormControl>
          <FormLabel>添加流程步骤</FormLabel>
          <Select
            placeholder="选择一个步骤"
            onChange={(event) => {
              const value = event.target.value;
              if (!value) return;
              setWorkflowSteps((prev) => [...prev, value]);
            }}
          >
            {steps.map((step) => (
              <option key={step.id} value={step.id}>
                {step.label}
              </option>
            ))}
          </Select>
        </FormControl>
        <HStack spacing={2} wrap="wrap">
          {workflowSteps.length === 0 && <Text color="gray.400">尚未添加步骤</Text>}
          {workflowSteps.map((item, index) => {
            const info = steps.find((step) => step.id === item);
            return (
              <Badge key={`${item}-${index}`} colorScheme="purple">
                {info?.label ?? item}
              </Badge>
            );
          })}
        </HStack>
        <Button
          colorScheme="blue"
          alignSelf="flex-start"
          isLoading={createMutation.isPending}
          onClick={handleCreateWorkflow}
        >
          保存为流程
        </Button>
      </Stack>

      <Heading size="sm" mb={2}>
        流程列表
      </Heading>
      {isLoading && <Text color="gray.500">加载中...</Text>}
      {!isLoading && workflows.length === 0 && (
        <Text color="gray.500">暂无流程，请先创建一个流程。</Text>
      )}
      <Stack spacing={3} mt={3}>
        {workflows.map((workflow) => (
          <WorkflowCard
            key={workflow.id}
            workflow={workflow}
            isPublishing={publishMutation.isPending}
            onPublish={(id) => publishMutation.mutate(id)}
          />
        ))}
      </Stack>
    </Box>
  );
}

function WorkflowCard({
  workflow,
  isPublishing,
  onPublish,
}: {
  workflow: WorkflowDefinition;
  isPublishing: boolean;
  onPublish: (id: string) => void;
}) {
  const steps = Array.isArray(workflow.blueprint?.steps)
    ? (workflow.blueprint.steps as Array<{ name?: string; type?: string }>)
    : [];

  return (
    <Box borderWidth="1px" borderRadius="md" p={3}>
      <Flex justify="space-between" align="center">
        <Box>
          <Heading size="sm">{workflow.name}</Heading>
          <Text fontSize="sm" color="gray.500">
            v{workflow.version} · {workflow.description || "无描述"}
          </Text>
        </Box>
        <HStack>
          <Badge colorScheme={workflow.published ? "green" : "yellow"}>
            {workflow.published ? "已发布" : "草稿"}
          </Badge>
          {!workflow.published && (
            <IconButton
              size="sm"
              aria-label="发布流程"
              icon={<CheckIcon />}
              isLoading={isPublishing}
              onClick={() => onPublish(workflow.id)}
            />
          )}
        </HStack>
      </Flex>
      {steps.length > 0 && (
        <HStack spacing={2} mt={2} wrap="wrap">
          {steps.map((step, index) => (
            <Badge key={`${workflow.id}-${index}`} colorScheme="purple" variant="subtle">
              {step.name ?? step.type ?? `步骤${index + 1}`}
            </Badge>
          ))}
        </HStack>
      )}
    </Box>
  );
}
