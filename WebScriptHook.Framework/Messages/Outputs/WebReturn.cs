﻿using Newtonsoft.Json;

namespace WebScriptHook.Framework.Messages.Outputs
{
    /// <summary>
    /// This message is sent to server as a response to a request
    /// </summary>
    class WebReturn : WebOutput
    {
        const char HEADER_RETURN = 'r';

        [JsonConstructor]
        public WebReturn(object Data, string UID)
            : base(HEADER_RETURN, new object[] { Data }, UID)
        { }
    }
}