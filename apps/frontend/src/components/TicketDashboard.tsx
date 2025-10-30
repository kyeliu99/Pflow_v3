import { useMemo } from "react";
import {
  Badge,
  Box,
  Flex,
  Heading,
  Progress,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
} from "@chakra-ui/react";

const tickets = [
  { id: "TK-1001", status: "进行中", workflow: "入职审批", assignee: "Zhang" },
  { id: "TK-1002", status: "已完成", workflow: "费用报销", assignee: "Li" },
  { id: "TK-1003", status: "待处理", workflow: "访问申请", assignee: "王" },
];

export default function TicketDashboard() {
  const metrics = useMemo(() => {
    const total = tickets.length;
    const done = tickets.filter((ticket) => ticket.status === "已完成").length;
    return { total, done, percent: total === 0 ? 0 : Math.round((done / total) * 100) };
  }, []);

  return (
    <Box borderWidth="1px" borderRadius="md" p={4} bg="white" shadow="sm">
      <Flex justify="space-between" align="center" mb={4}>
        <Heading size="md">工单看板</Heading>
        <Badge colorScheme="green">完成率 {metrics.percent}%</Badge>
      </Flex>
      <Progress value={metrics.percent} mb={4} colorScheme="green" />
      <Table size="sm">
        <Thead>
          <Tr>
            <Th>工单号</Th>
            <Th>流程</Th>
            <Th>处理人</Th>
            <Th>状态</Th>
          </Tr>
        </Thead>
        <Tbody>
          {tickets.map((ticket) => (
            <Tr key={ticket.id}>
              <Td>{ticket.id}</Td>
              <Td>{ticket.workflow}</Td>
              <Td>{ticket.assignee}</Td>
              <Td>
                <Badge>{ticket.status}</Badge>
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
      <Text mt={4} color="gray.500" fontSize="sm">
        集成 OpenAPI 或 npm SDK 以加载实时数据。
      </Text>
    </Box>
  );
}
