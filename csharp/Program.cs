using AngleSharp;
using AngleSharp.Dom;
using System;
using System.IO;
using System.Linq;
using System.Net.Http;
using System.Text.RegularExpressions;
using System.Threading;
using System.Threading.Tasks;

namespace ikkdm_fch
{
    class Program : IDisposable
    {
        private static readonly Regex ScriptRegex = new Regex("<IMG SRC='(?<url>.*?)'>");
        private static string baseLocation = "./download";
        private static HttpClient httpClient = new HttpClient();

        private static int count;
        private static int failCount;

        static void Main(string[] args)
        {
            Console.Write("数据ikkdm的移动版链接 >>> ");
            var homeUrl = Console.ReadLine();
            Console.Write("数据最大请求数(默认10，-1不限制): ");
            if (!int.TryParse(Console.ReadLine(), out var maxConnection))
            {
                maxConnection = 10;
            }

            var document = OpenAsync(homeUrl).Result;

            var title = document.QuerySelector("#comicName").TextContent;

            var episodes = document
                .QuerySelector("#list")
                .QuerySelectorAll("li > a[href]")
                .Select(e => new Episode { Link = e.GetAttribute("href"), Title = e.TextContent })
                .ToList();

            if (maxConnection <= 0 || maxConnection > episodes.Count)
            {
                maxConnection = episodes.Count;
            }
            Console.WriteLine("最大请求数: " + maxConnection.ToString());

            if (!Directory.Exists(baseLocation))
            {
                Directory.CreateDirectory(baseLocation);
            }

            baseLocation = Path.Join(baseLocation, title);

            if (!Directory.Exists(baseLocation))
            {
                Directory.CreateDirectory(baseLocation);
            }
            Console.WriteLine("-------- 开始下载 --------");
            Console.WriteLine("-------- 一共" + episodes.Count + "话 --------");
            using (var pool = new Semaphore(maxConnection, maxConnection))
            {
                for (int i = 0; i < episodes.Count; i++)
                {
                    pool.WaitOne();
                    Task.Factory.StartNew((index) =>
                    {
                        var idx = (int)index;
                        Console.WriteLine("开始下载第" + (idx + 1).ToString() + "话");
                        FchEpisode(episodes[idx]);
                        Interlocked.Increment(ref count);
                        Console.WriteLine("已保存" + count + "话，保存" + failCount + "张图片失败");
                        pool.Release();
                    }, i);
                }

                // 等待所有下载完成
                while (count < episodes.Count)
                {
                    Thread.Yield();
                }
                Console.WriteLine("-------- 下载完成 --------");
            }
        }

        private static void FchEpisode(Episode episode)
        {
            var url = "http://m.ikkdm.com" + episode.Link;

            var box = OpenAsync(url).Result.QuerySelector("div.classBox.autoHeight");

            var info = box.QuerySelectorAll("div.bottom ul.subNav li").Skip(1).Take(1).First().TextContent;
            var episodeCount = int.Parse(info.Split('/')[1]);

            // 创建当前话的下载目录
            var location = Path.Join(baseLocation, episode.Title);
            if (!Directory.Exists(location))
            {
                Directory.CreateDirectory(location);
            }

            var baseUrl = url.Substring(0, url.LastIndexOf('/') + 1);
            var tasks = new Task[episodeCount];
            for (int i = 1; i <= episodeCount; i++)
            {
                var task = Task.Factory.StartNew(
                    (u) => FchImage(episode.Title, u as string).Wait(),
                    baseUrl + i.ToString() + ".htm",
                    TaskCreationOptions.LongRunning
                );
                tasks[i - 1] = task;
            }
            Task.WaitAll(tasks);
            Console.WriteLine("{{ " + episode.Title + " }}下载完成");
        }

        private static async Task FchImage(string title, string url)
        {
            IElement box = null;
            try
            {
                box = (await OpenAsync(url)).QuerySelector("div.classBox.autoHeight");
            }
            catch (Exception)
            {
                Interlocked.Increment(ref failCount);
                Console.WriteLine("请求 {{ " + url + " }} 时遇到错误");
                return;
            }

            var script = box.QuerySelector("script[language=javascript]").TextContent;
            var matchedUrl = ScriptRegex.Match(script).Groups["url"].Value.Replace("\"+m2007+\"", "http://m8.1whour.com/");

            var i1 = url.LastIndexOf('/');
            var i2 = url.LastIndexOf('.');
            var imgName = url.Substring(i1 + 1, i2 - i1 - 1);
            await SaveImage(matchedUrl, title, imgName);
        }

        private static async Task SaveImage(string url, string title, string imgName)
        {
            var stream = await httpClient.GetStreamAsync(url);
            var ext = Path.GetFileName(url);
            var path = Path.Join(baseLocation, title, imgName + "_" + ext);
            using (var fs = File.Create(path, 1024, FileOptions.Asynchronous))
            {
                await stream.CopyToAsync(fs);
            }
        }

        private static async Task<IDocument> OpenAsync(string url)
        {
            var config = Configuration.Default.WithCulture("zh-CN").WithLocaleBasedEncoding();
            var context = BrowsingContext.New(config);
            return await context.OpenAsync(res =>
            {
                var stream = httpClient.GetStreamAsync(url).Result;
                res.Content(stream);
            });
        }

        public void Dispose()
        {
            httpClient.Dispose();
        }
    }
}
