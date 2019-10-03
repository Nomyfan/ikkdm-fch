using System.Threading.Tasks;

namespace ikkdm_fch
{
    public static class TaskEx
    {
        public static Task WhenAll(this Task[] tasks)
        {
            return Task.WhenAll(tasks);
        }
    }
}