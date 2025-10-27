FROM scratch 
ARG TARGETPLATFORM
WORKDIR /app
COPY $TARGETPLATFORM/down-pvscope ./main
CMD ["./main"]

