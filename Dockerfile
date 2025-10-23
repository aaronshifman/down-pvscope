FROM scratch 

WORKDIR /app
COPY bin/main ./main
CMD ["./main"]

