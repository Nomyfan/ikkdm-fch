using System.Text;
using AngleSharp.Browser;
using AngleSharp.Html.Dom;

namespace ikkdm_fch
{
    class Utf8MetaHandler : IMetaHandler
    {
        public void HandleContent(IHtmlMetaElement element)
        {
            element.Owner.Source.CurrentEncoding = Encoding.UTF8;
        }
    }
}