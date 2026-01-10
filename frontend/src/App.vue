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
  <div class="flex h-screen w-screen bg-[#1C1C1C] text-gray-300 font-sans overflow-hidden">
    <Sidebar />
    <main class="flex-1 flex flex-col min-w-0">
      <RequestPanel @send="onExecuteRequest" />
      <ResponsePanel :data="responseData" />
    </main>
  </div>
</template>