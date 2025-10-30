import { useState } from "react";
import {
  Badge,
  Box,
  Button,
  Flex,
  Heading,
  HStack,
  Select,
  Text,
  VStack,
} from "@chakra-ui/react";

const steps = [
  { id: "form", label: "表单" },
  { id: "approval", label: "人工审批" },
  { id: "automation", label: "自动化执行" },
  { id: "notification", label: "通知" },
];

export default function WorkflowDesigner() {
  const [workflow, setWorkflow] = useState<string[]>([]);

  return (
    <Box borderWidth="1px" borderRadius="md" p={4} bg="white" shadow="sm">
      <Flex justify="space-between" align="center" mb={4}>
        <Heading size="md">可视化流程编排</Heading>
        <Button colorScheme="blue" size="sm" onClick={() => setWorkflow([])}>
          重置
        </Button>
      </Flex>
      <Text mb={2} color="gray.600">
        选择步骤以编排流程（支持人工+自动化）。
      </Text>
      <VStack align="stretch" spacing={3}>
        <Select
          placeholder="添加流程步骤"
          onChange={(event) => {
            const value = event.target.value;
            if (!value) return;
            setWorkflow((prev) => [...prev, value]);
          }}
        >
          {steps.map((step) => (
            <option key={step.id} value={step.id}>
              {step.label}
            </option>
          ))}
        </Select>
        <HStack spacing={2} wrap="wrap">
          {workflow.map((item, index) => {
            const info = steps.find((step) => step.id === item);
            return (
              <Badge key={`${item}-${index}`} colorScheme="purple">
                {info?.label ?? item}
              </Badge>
            );
          })}
          {workflow.length === 0 && <Text color="gray.400">尚未添加步骤</Text>}
        </HStack>
      </VStack>
    </Box>
  );
}
