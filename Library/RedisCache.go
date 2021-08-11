package Library

import (
	"context"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

var (
	Rdb    *redis.Client
	Rdberr error
)

func Rdlink() (Rdb *redis.Client, Rdberr error) {
	conf, Rdberr := GetConf()
	if Rdberr != nil {
		return
	}

	Rdb = redis.NewClient(&redis.Options{
		Addr:     conf.Redis.RedisHost + ":" + conf.Redis.RedisPort,
		Password: conf.Redis.RedisPwd, // no password set
		DB:       conf.Redis.RedisDb,  // use default DB
	})
	_, Rdberr = Rdb.Ping(ctx).Result()
	return
}

/**
 * 返回redis对象
 */
//public function rdlink() {
//    if (empty($this->config['REDIS_SERVER']) || empty($this->config['REDIS_POST']) || !isset($this->config['REDIS_KEY_PREFIX'])) {
//        echo "请配置Redis服务器信息";
//        return false;
//    }
//    if ($this->links !== null && $this->links->ping() != '+PONG') {
//        $this->links = null;
//    }
//    if ($this->links === null) {
//        $this->links = new \Redis();
//        rdb := redis.NewClient(&redis.Options{
//            Addr:     "localhost:6379",
//            Password: "", // no password set
//            DB:       0,  // use default DB
//        })
//        $res = $this->links->connect($this->config['REDIS_SERVER'], $this->config['REDIS_POST']);
//        if ($res === false) {
//            $this->links = null;
//            return false;
//        }
//        if (!empty($this->config['pwd'])) {
//            $this->links->auth($this->config['pwd']);
//        }
//    }
//
//    $this->prefix = $this->config['REDIS_KEY_PREFIX'];
//    $this->strReplace($this->prefix);
//
//    return $this->links;
//}
