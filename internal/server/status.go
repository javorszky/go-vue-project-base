package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
)

type statusResponse struct {
	Status    string `json:"status"`
	GitSHA    string `json:"git_sha"`
	BuildTime string `json:"build_time"`
}

func statusHandler(info BuildInfo) echo.HandlerFunc {
	resp := statusResponse{
		Status:    "ok",
		GitSHA:    info.GitSHA,
		BuildTime: info.BuildTime,
	}
	return func(c *echo.Context) error {
		if err := c.JSON(http.StatusOK, resp); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
		return nil
	}
}
