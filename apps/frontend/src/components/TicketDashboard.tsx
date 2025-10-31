import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  AlertIcon,
  Badge,
  Box,
  Button,
  FormControl,
  FormLabel,
  Heading,
  Input,
  Select,
  SimpleGrid,
  Stack,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useToast,
} from "@chakra-ui/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CreateTicketPayload,
  ItemResponse,
  getTicketQueueMetrics,
  getTicketSubmission,
  listForms,
  listTickets,
  listUsers,
  resolveTicket,
  submitTicket,
  Ticket,
  TicketQueueMetrics,
  TicketSubmission,
} from "../lib/api";

const statusLabels: Record<string, string> = {
  open: "待处理",
  in_progress: "进行中",
  resolved: "已完成",
  cancelled: "已取消",
};

const statusColors: Record<string, string> = {
  open: "yellow",
  in_progress: "blue",
  resolved: "green",
  cancelled: "gray",
};

const priorityOptions = [
  { value: "low", label: "低" },
  { value: "medium", label: "中" },
  { value: "high", label: "高" },
];

function isFinalSubmissionStatus(status?: string) {
  return status === "completed" || status === "failed";
}

export default function TicketDashboard() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [title, setTitle] = useState("");
  const [formId, setFormId] = useState("");
  const [assigneeId, setAssigneeId] = useState("");
  const [priority, setPriority] = useState("medium");
  const [activeSubmissionId, setActiveSubmissionId] = useState<string | null>(null);
  const [lastSubmissionStatus, setLastSubmissionStatus] = useState<string | null>(null);

  const { data: ticketsData, isLoading: ticketsLoading, isError } = useQuery({
    queryKey: ["tickets"],
    queryFn: () => listTickets(),
  });
  const { data: formsData } = useQuery({
    queryKey: ["forms"],
    queryFn: () => listForms(),
  });
  const { data: usersData } = useQuery({
    queryKey: ["users"],
    queryFn: () => listUsers(),
  });
  const { data: queueMetricsData } = useQuery({
    queryKey: ["ticketQueueMetrics"],
    queryFn: () => getTicketQueueMetrics(),
    refetchInterval: 5000,
  });

  const tickets = ticketsData?.data ?? [];
  const forms = formsData?.data ?? [];
  const users = usersData?.data ?? [];
  const queueMetrics: TicketQueueMetrics | undefined = queueMetricsData;

  const submissionQuery = useQuery<ItemResponse<TicketSubmission>, Error>({
    queryKey: ["ticketSubmission", activeSubmissionId],
    queryFn: () => getTicketSubmission(activeSubmissionId as string),
    enabled: Boolean(activeSubmissionId),
  });
  const { refetch: refetchSubmission } = submissionQuery;

  const submitMutation = useMutation({
    mutationFn: (payload: CreateTicketPayload & { clientReference?: string }) => submitTicket(payload),
    onSuccess: (response) => {
      const submission = response.data;
      setActiveSubmissionId(submission.id);
      setLastSubmissionStatus(submission.status);
      queryClient.invalidateQueries({ queryKey: ["ticketQueueMetrics"] });

      if (submission.status === "completed") {
        toast({ status: "success", title: "工单已创建" });
        queryClient.invalidateQueries({ queryKey: ["tickets"] });
        queryClient.invalidateQueries({ queryKey: ["overview"] });
        resetForm();
        setActiveSubmissionId(null);
      } else {
        toast({ status: "info", title: "工单已进入队列", description: "系统正在后台处理，请稍候。" });
      }
    },
    onError: () => {
      toast({ status: "error", title: "提交工单失败", description: "请稍后再试" });
    },
  });

  const resolveMutation = useMutation({
    mutationFn: resolveTicket,
    onSuccess: () => {
      toast({ status: "success", title: "工单已完成" });
      queryClient.invalidateQueries({ queryKey: ["tickets"] });
      queryClient.invalidateQueries({ queryKey: ["overview"] });
    },
    onError: () => {
      toast({ status: "error", title: "更新工单状态失败" });
    },
  });

  const assigneeMap = useMemo(() => {
    const map = new Map<string, string>();
    users.forEach((user) => map.set(user.id, user.name));
    return map;
  }, [users]);

  const formMap = useMemo(() => {
    const map = new Map<string, string>();
    forms.forEach((form) => map.set(form.id, form.name));
    return map;
  }, [forms]);

  useEffect(() => {
  const submission: TicketSubmission | undefined = submissionQuery.data?.data;
    if (!submission) {
      return;
    }

    if (submission.status !== lastSubmissionStatus) {
      setLastSubmissionStatus(submission.status);

      if (submission.status === "completed") {
        toast({ status: "success", title: "工单已创建" });
        queryClient.invalidateQueries({ queryKey: ["tickets"] });
        queryClient.invalidateQueries({ queryKey: ["overview"] });
        queryClient.invalidateQueries({ queryKey: ["ticketQueueMetrics"] });
        resetForm();
        setActiveSubmissionId(null);
      } else if (submission.status === "failed") {
        toast({ status: "error", title: "工单创建失败", description: submission.errorMessage ?? "请稍后重试" });
      }
    }
  }, [submissionQuery.data, lastSubmissionStatus, queryClient, toast]);

  const submissionStatus = submissionQuery.data?.data.status ?? lastSubmissionStatus ?? undefined;
  const submissionInFlight = Boolean(
    activeSubmissionId && (!submissionStatus || !isFinalSubmissionStatus(submissionStatus))
  );

  useEffect(() => {
    if (!submissionInFlight || !activeSubmissionId) {
      return;
    }

    const timer = setInterval(() => {
      refetchSubmission();
    }, 2000);

    return () => clearInterval(timer);
  }, [submissionInFlight, activeSubmissionId, refetchSubmission]);

  const handleCreateTicket = () => {
    if (!title.trim()) {
      toast({ status: "warning", title: "请输入工单标题" });
      return;
    }
    if (!formId) {
      toast({ status: "warning", title: "请选择关联表单" });
      return;
    }

    setActiveSubmissionId(null);
    setLastSubmissionStatus(null);

    const payload: CreateTicketPayload & { clientReference: string } = {
      title: title.trim(),
      formId,
      assigneeId: assigneeId || undefined,
      priority,
      clientReference: crypto.randomUUID(),
    };

    submitMutation.mutate(payload);
  };

  const resetForm = () => {
    setTitle("");
    setFormId("");
    setAssigneeId("");
    setPriority("medium");
  };

  return (
    <Box borderWidth="1px" borderRadius="md" p={4} bg="white" shadow="sm">
      <Heading size="md" mb={4}>
        工单管理
      </Heading>

      <SimpleGrid columns={{ base: 1, md: 4 }} spacing={3} mb={6}>
        <Stat>
          <StatLabel>队列待处理</StatLabel>
          <StatNumber>{queueMetrics?.pending ?? 0}</StatNumber>
          <StatHelpText>等待分发</StatHelpText>
        </Stat>
        <Stat>
          <StatLabel>处理中</StatLabel>
          <StatNumber>{queueMetrics?.processing ?? 0}</StatNumber>
          <StatHelpText>工作进度</StatHelpText>
        </Stat>
        <Stat>
          <StatLabel>近期开结</StatLabel>
          <StatNumber>{queueMetrics?.completed ?? 0}</StatNumber>
          <StatHelpText>累计完成</StatHelpText>
        </Stat>
        <Stat>
          <StatLabel>最久等待</StatLabel>
          <StatNumber>{queueMetrics?.oldestPendingSeconds ?? 0}s</StatNumber>
          <StatHelpText>队列最长排队</StatHelpText>
        </Stat>
      </SimpleGrid>

      <Stack spacing={4} mb={6}>
        <FormControl>
          <FormLabel>工单标题</FormLabel>
          <Input
            placeholder="例如：入职流程审批"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
          />
        </FormControl>
        <FormControl>
          <FormLabel>关联表单</FormLabel>
          <Select placeholder="选择一个表单" value={formId} onChange={(event) => setFormId(event.target.value)}>
            {forms.map((form) => (
              <option key={form.id} value={form.id}>
                {form.name}
              </option>
            ))}
          </Select>
        </FormControl>
        <FormControl>
          <FormLabel>指派给</FormLabel>
          <Select
            placeholder="可选"
            value={assigneeId}
            onChange={(event) => setAssigneeId(event.target.value)}
          >
            {users.map((user) => (
              <option key={user.id} value={user.id}>
                {user.name}
              </option>
            ))}
          </Select>
        </FormControl>
        <FormControl>
          <FormLabel>优先级</FormLabel>
          <Select value={priority} onChange={(event) => setPriority(event.target.value)}>
            {priorityOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </Select>
        </FormControl>
        <Button
          colorScheme="blue"
          alignSelf="flex-start"
          isLoading={submitMutation.isPending || submissionInFlight}
          onClick={handleCreateTicket}
          isDisabled={forms.length === 0}
        >
          新建工单
        </Button>
        {submissionInFlight && (
          <Alert status="info" borderRadius="md">
            <AlertIcon /> 工单正在队列中排队处理，请勿重复提交。
          </Alert>
        )}
        {forms.length === 0 && (
          <Alert status="warning" borderRadius="md">
            <AlertIcon /> 创建工单前，请先创建一个表单。
          </Alert>
        )}
      </Stack>

      <Heading size="sm" mb={2}>
        工单列表
      </Heading>
      {isError && (
        <Alert status="error" borderRadius="md" mb={3}>
          <AlertIcon /> 无法加载工单数据，请检查工单服务。
        </Alert>
      )}
      {ticketsLoading && <Text color="gray.500">加载中...</Text>}
      {!ticketsLoading && tickets.length === 0 && <Text color="gray.500">暂无工单。</Text>}

      {tickets.length > 0 && (
        <Table size="sm" mt={3}>
          <Thead>
            <Tr>
              <Th>标题</Th>
              <Th>表单</Th>
              <Th>指派</Th>
              <Th>状态</Th>
              <Th>操作</Th>
            </Tr>
          </Thead>
          <Tbody>
            {tickets.map((ticket: Ticket) => (
              <Tr key={ticket.id}>
                <Td>{ticket.title}</Td>
                <Td>{formMap.get(ticket.formId) ?? ticket.formId}</Td>
                <Td>{assigneeMap.get(ticket.assigneeId) ?? "未指派"}</Td>
                <Td>
                  <Badge colorScheme={statusColors[ticket.status] ?? "gray"}>
                    {statusLabels[ticket.status] ?? ticket.status}
                  </Badge>
                </Td>
                <Td>
                  {ticket.status !== "resolved" && (
                    <Button
                      size="xs"
                      variant="outline"
                      isLoading={resolveMutation.isPending}
                      onClick={() => resolveMutation.mutate(ticket.id)}
                    >
                      标记完成
                    </Button>
                  )}
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
    </Box>
  );
}
