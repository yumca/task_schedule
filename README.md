    # task_schedule
##########################################
===
    # 定时任务
    # Conf/cron.json
    task_key=sync_tasks  //投递什么任务  sync_tasks同步任务  async_tasks异步任务
    TaskType=PHP  //执行什么程序脚本
    Expire=1  //执行过期时间  /秒
    run_time=1 * * * *  //定时时间
    script=D:\WWW\BuGu\points\index.php task/TaskTest execute   //执行脚本地址和参数  绝对路径
    Param=   //执行参数  暂时无用
    UniqueTaskId=task_test   //执行唯一key

    数据示例：
    异步
    [
        {
            "TaskKey": "async_tasks",
            "TaskType": "PHP",
            "Expire": "1",
            "RunTime": "* * * * *",
            "Script": "D:\\WWW\\BuGu\\points\\index.php task/TaskTest execute",
            "Param": "",
            "UniqueTaskId": "task_test"
        }
    ]
    同步
    [
        {
            "TaskKey": "sync_tasks",
            "TaskType": "PHP",
            "Expire": "1",
            "RunTime": "* * * * *",
            "Script": "D:\\WWW\\BuGu\\points\\index.php task/TaskTest execute",
            "Param": "",
            "UniqueTaskId": "task_test"
        }
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