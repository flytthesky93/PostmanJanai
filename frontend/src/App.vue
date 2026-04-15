<script setup>
import { ref } from 'vue'
import Sidebar from './components/Sidebar.vue'
import RequestPanel from './components/RequestPanel.vue'
import ResponsePanel from './components/ResponsePanel.vue'

const responseData = ref('')

const onExecuteRequest = (payload) => {
  console.log("Thực thi:", payload)
  // Giả lập kết quả trả về
  responseData.value = JSON.stringify({
    message: `Đã gửi ${payload.method} tới ${payload.url}`,
    status: 200,
    timestamp: new Date().toISOString()
  }, null, 2)
}
</script>

<template>
  <!-- No position:fixed on root — fills #app (absolute inset 0). Flex row + fixed sidebar width. -->
  <div
    class="font-sans text-gray-300"
    style="
      width: 100%;
      height: 100%;
      min-width: 0;
      min-height: 0;
      overflow: hidden;
      display: flex;
      flex-direction: row;
      align-items: stretch;
      background: #1c1c1c;
    "
  >
    <div
      style="
        width: 256px;
        min-width: 256px;
        max-width: 256px;
        flex: 0 0 256px;
        height: 100%;
        min-height: 0;
        overflow: hidden;
        background: #212121;
        border-right: 1px solid #2a2a2a;
      "
    >
      <Sidebar />
    </div>
    <main
      class="flex min-h-0 min-w-0 flex-col overflow-hidden"
      style="flex: 1 1 0; min-width: 0; min-height: 0; display: flex; flex-direction: column; height: 100%"
    >
      <RequestPanel @send="onExecuteRequest" />
      <ResponsePanel :data="responseData" />
    </main>
  </div>
</template>