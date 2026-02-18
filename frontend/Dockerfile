# Stage 1 — build
FROM node:18 AS build

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем package.json и package-lock.json для кеширования npm install
COPY package.json package-lock.json ./

# Устанавливаем зависимости
RUN npm ci --legacy-peer-deps

# Копируем весь проект
COPY . .

# Собираем приложение
RUN npm run build

# Stage 2 — serve with nginx
FROM nginx:alpine

# Удаляем дефолтный контент nginx
RUN rm -rf /usr/share/nginx/html/*

# Копируем собранные файлы из build stage
COPY --from=build /app/dist /usr/share/nginx/html

# Кастомный конфиг nginx: SPA + proxy на backend контейнер
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Открываем порт 80
EXPOSE 80

# Запуск nginx в форграунд режиме
CMD ["nginx", "-g", "daemon off;"]
