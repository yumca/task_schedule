    # task_schedule
##########################################
    # 定时任务
    # Conf/cron.ini
    task_key=sync_tasks  //投递什么任务  sync_tasks同步任务  async_tasks异步任务
    run_time=1 * * * *  //定时时间
    script=/var/www/points/index.php   //框架入口文件地址  绝对路径
    controller=api/AccumulatePay   //执行的控制器地址
    function=execute   //执行的方法
    param=   //需要注入的参数  字符串并需要urlencode

    数据示例：
    异步
    [
        'task_key'=>'sync_tasks'
        'script'=>'/var/www/points/index.php',
        'controller'=>'api/AccumulatePay',
        'function'=>'execute',
        'param'=>[
            '字段1'=>'字段1值'
        ]
    ]
    同步
    [
        'task_key'=>'async_tasks'
        'script'=>'/var/www/points/index.php',
        'controller'=>'api/AccumulatePay',
        'function'=>'execute',
        'param'=>[
            '字段1'=>'字段1值'
        ],
    ]

    #任务脚本执行返回结构
    code=succ  //执行返回code  succ成功   fail失败  or 其他状态
    msg=message  //执行返回信息
    data=[]   //返回数据  array
    [
        'code'=>'async_tasks'
        'msg'=>'/var/www/points/index.php',
        'data'=>[
            '字段1'=>'字段1值'
        ],
    ]
##########################################
=====
进程监控redis结构
worker数据结构:
    key:
        Task:Worker:workerId
    name:
        worker_id  ,task_key  ,status                         ,running_task        ,restart_num   ,run_time         ,run_return
    value:
        进程id     ,进程key    ,进程状态 running运行 stop停止  ,正在执行的任务数量    ,进程重启次数   ,进程运行开始时间  ,执行返回
taskworker数据结构:
    key:
        Task:taskWorker:workerId
    name:
        worker_id  ,task_key  ,status                         ,running_task        ,restart_num   ,run_time
    value:
        进程id     ,进程key    ,进程状态 running运行 stop停止  ,正在执行的任务数量    ,进程重启次数   ,进程运行开始时间
正在执行的任务数据结构:
    key:
        Task:List:workerId:uniqueTaskId
    name:
        worker_id  ,task_key    ,start_runtime    ,script                     ,controller       ,function   ,param      ,unique_task_id
    value:
        进程id     ,任务队列key  ,任务运行开始时间  ,框架入口文件地址  绝对路径  ,执行的控制器地址  ,执行的方法  ,注入的参数  ,执行任务唯一id
执行失败的任务数据结构(存储在正式数据库task_schedule的fail_log):
    key:
        Task:Fail:workerId:uniqueTaskId
    name:
        worker_id  ,task_key    ,start_runtime    ,end_runtime      ,script                     ,controller       ,function   ,param      ,unique_task_id
    value:
        进程id     ,任务队列key  ,任务运行开始时间  ,任务运行结束时间  ,框架入口文件地址  绝对路径  ,执行的控制器地址  ,执行的方法  ,注入的参数  ,执行任务唯一id
=====