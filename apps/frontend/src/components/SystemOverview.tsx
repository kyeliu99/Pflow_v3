import {
  Alert,
  AlertIcon,
  Box,
  Heading,
  SimpleGrid,
  Skeleton,
  Stat,
  StatLabel,
  StatNumber,
  Text,
} from "@chakra-ui/react";
import { useQuery } from "@tanstack/react-query";
import { getOverview } from "../lib/api";

export default function SystemOverview() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["overview"],
    queryFn: getOverview,
    refetchInterval: 15000,
  });

  if (isLoading) {
    return (
      <Box borderWidth="1px" borderRadius="md" p={4} bg="white" shadow="sm">
        <Heading size="md" mb={4}>
          系统概览
        </Heading>
        <SimpleGrid columns={{ base: 1, md: 4 }} spacing={4}>
          {[0, 1, 2, 3].map((key) => (
            <Skeleton key={key} height="80px" borderRadius="md" />
          ))}
        </SimpleGrid>
      </Box>
    );
  }

  if (isError || !data) {
    return (
      <Box borderWidth="1px" borderRadius="md" p={4} bg="white" shadow="sm">
        <Heading size="md" mb={4}>
          系统概览
        </Heading>
        <Alert status="error" borderRadius="md">
          <AlertIcon /> 无法加载概览数据，请确认网关与各微服务是否正常运行。
        </Alert>
      </Box>
    );
  }

  const overview = data.data;

  return (
    <Box borderWidth="1px" borderRadius="md" p={4} bg="white" shadow="sm">
      <Heading size="md" mb={4}>
        系统概览
      </Heading>
      <SimpleGrid columns={{ base: 2, md: 4 }} spacing={4}>
        <StatCard label="表单" value={overview.forms.total} />
        <StatCard
          label="工单"
          value={overview.tickets.total}
          helperText={`处理中 ${overview.tickets.byStatus["in_progress"] ?? 0}`}
        />
        <StatCard label="用户" value={overview.users.total} />
        <StatCard
          label="流程"
          value={overview.workflows.total}
          helperText={`已发布 ${overview.workflows.published}`}
        />
      </SimpleGrid>
      <Text color="gray.500" fontSize="sm" mt={3}>
        数据由 API 网关实时聚合，可根据需要扩展更多指标。
      </Text>
    </Box>
  );
}

function StatCard({
  label,
  value,
  helperText,
}: {
  label: string;
  value: number;
  helperText?: string;
}) {
  return (
    <Stat borderWidth="1px" borderRadius="md" p={3}>
      <StatLabel>{label}</StatLabel>
      <StatNumber>{value}</StatNumber>
      {helperText && (
        <Text fontSize="sm" color="gray.500">
          {helperText}
        </Text>
      )}
    </Stat>
  );
}
