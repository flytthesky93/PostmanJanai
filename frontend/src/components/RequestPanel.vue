<script setup>
import { ref } from 'vue'

// Định nghĩa sự kiện để báo cho cha (App.vue) biết khi nào nhấn Send
const emit = defineEmits(['send'])

const url = ref('')
const method = ref('GET')

const handleSend = () => {
  emit('send', { method: method.value, url: url.value })
}
</script>

<template>
  <div class="p-4 border-b border-gray-800 bg-[#212121] flex gap-2 items-center">
    <select v-model="method" class="bg-gray-800 px-3 py-1.5 rounded text-green-500 font-bold text-sm outline-none border border-gray-700">
      <option>GET</option>
      <option>POST</option>
      <option>PUT</option>
      <option>DELETE</option>
    </select>
    <input
        v-model="url"
        type="text"
        class="flex-1 bg-gray-900 border border-gray-700 rounded px-3 py-1.5 outline-none focus:border-orange-500 text-sm text-gray-200"
        placeholder="https://api.example.com/v1/data"
    />
    <button @click="handleSend" class="bg-orange-600 hover:bg-orange-700 text-white px-6 py-1.5 rounded font-bold text-sm transition-all active:scale-95">
      Send
    </button>
  </div>
</template>