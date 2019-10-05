using System.Text;
using System.Threading.Tasks;

namespace ikkdm_fch
{
    public static class Exts
    {
        public static Task WhenAll(this Task[] tasks)
        {
            return Task.WhenAll(tasks);
        }

        public static string GbkByteArrayToUtf8String(this byte[] gbk)
        {
            return Encoding.GetEncoding("GB18030").GetString(gbk);
        }
    }
}