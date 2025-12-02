import React, { useEffect, useState } from "react";
import { useAppSelector, useAppDispatch } from "../../app/hooks";
import { useNavigate } from "react-router-dom";

export function OAuthTwitch() {
	const navigate = useNavigate();
	const [oAuthSuccessObj, setOAuthSuccessObj] = useState<{
		access_token: string;
		scope: string;
		state?: string;
		token_type: string;
	} | null>(null);
	const [errorObj, setErrorObj] = useState<{
		errorMsg: string;
		error_description: string;
	} | null>(null);

	useEffect(() => {
		// check initial state
		/*
			example fragment: #access_token=73d0f8mkabpbmjp921asv2jaidwxn&scope=channel%3Amanage%3Apolls+channel%3Aread%3Apolls&state=c3ab8aa609ea11e793ae92361f002671&token_type=bearer
			example error: ?error=redirect_mismatch&error_description=Parameter+redirect_uri+does+not+match+registered+URI
		*/
		if (window.location.search) {
			try {
				const queryParams = new URLSearchParams(window.location.search);
				if (queryParams.has("error") && queryParams.has("error_description")) {
					setErrorObj({
						errorMsg: queryParams.get("error") ?? "",
						error_description: queryParams.get("error_description") ?? "",
					});
					return;
				}
			} catch (e) {
				// do nothing
				return;
			}
		} else if (window.location.hash) {
			try {
				const hashParams = new URLSearchParams(
					window.location.hash.substring(1),
				);
				if (hashParams.has("access_token")) {
					const obj: {
						access_token: string;
						scope: string;
						state?: string;
						token_type: string;
					} = {
						access_token: hashParams.get("access_token") ?? "",
						scope: hashParams.get("access_token") ?? "",
						token_type: hashParams.get("token_type") ?? "",
					};
					if (hashParams.has("state")) {
						obj.state = hashParams.get("state") ?? "";
					}
					setOAuthSuccessObj(obj);
				}
			} catch (e) {
				return;
			}
		} else {
			navigate("/oauth/twitch-connect");
		}
	}, []);

	return (
		<div>
			{errorObj !== null ? (
				<>{JSON.stringify(errorObj)}</>
			) : oAuthSuccessObj !== null ? (
				<>{JSON.stringify(oAuthSuccessObj)}</>
			) : (
				<>default</>
			)}
		</div>
	);
}
