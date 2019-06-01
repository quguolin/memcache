
### 一：Error

#### `ERROR\r\n`
  ```
  客户端发送了一个不存在的命令
  ```

#### `CLIENT_ERROR\r\n`
  ```
  客户端发送了一个不符合协议的命令
  ```  
    
#### `SERVER_ERROR\r\n`
  ```
  服务的错误
  ```  
  
### 二：Storage commands
```
1.<command name> <key> <flags> <exptime> <bytes> [noreply]\r\n
```
```
2.cas <key> <flags> <exptime> <bytes> <cas unique> [noreply]\r\n
```

**command name is "set", "add", "replace", "append" or "prepend"**

#### `set`
  ```
  用于将 value(数据值) 存储在指定的 key(键) 中
  如果set的key已经存在，该命令可以更新该key所对应的原来的数据
  ```
  - STORED：保存成功后输出
  - ERROR：在保存失败后输出
#### `add`
  ```
  用于将 value(数据值) 存储在指定的 key(键) 中
  如果 add 的 key 已经存在，则不会更新数据(过期的 key 会更新)，之前的值将仍然保持相同，并且获得响应 NOT_STORED
  ``` 
  - STORED：保存成功后输出
  - NOT_STORED ：在保存失败后输出
#### `replace`
  ```
  用于向已存在 key(键) 的 value(数据值) 后面追加数据 
  如果 key 不存在，则替换失败，并且获得响应 NOT_STORED
  ```
  - STORED：保存成功后输出
  - NOT_STORED ：在保存失败后输出
#### `append`
  ```
  用于向已存在 key(键) 的 value(数据值) 后面追加数据 
  STORED：保存成功后输出
  ```
  - NOT_STORED：该键在 Memcached 上不存在
  - CLIENT_ERROR：执行错误  
#### `prepend`
  ```
  用于向已存在 key(键) 的 value(数据值) 前面追加数据 
  ```
   - STORED：保存成功后输出
   - NOT_STORED：该键在 Memcached 上不存在
   - CLIENT_ERROR：执行错误
#### `cas`
  ```
  命令用于执行一个"检查并设置"的操作它仅在当前客户端最后一次取值后，该key 对应的值没有被其他客户端修改的情况下， 才能够将值写入。
  检查是通过cas_token参数进行的， 这个参数是Memcach指定给已经存在的元素的一个唯一的64位值
  首先需要从 Memcached 服务商通过 gets 命令获取令牌（token），gets 返回64位的整型值非常像名称/值对的 "版本" 标识符
  ```
  - STORED：保存成功后输出
  - ERROR：保存出错或语法错误
  - EXISTS：在最后一次取值后另外一个用户也在更新该数据
  - NOT_FOUND：Memcached 服务上不存在该键值
