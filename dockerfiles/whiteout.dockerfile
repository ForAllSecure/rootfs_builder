FROM alpine
RUN mkdir /dir
RUN mkdir /dir2
RUN touch /dir/file1
RUN rm -r /dir
RUN touch /dir2/file2
