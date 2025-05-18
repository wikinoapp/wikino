# typed: true
# frozen_string_literal: true

class ApplicationController < ActionController::Base
  extend T::Sig

  private def render_404
    render(
      file: Rails.public_path.join("404.html"),
      status: :not_found,
      layout: false,
      content_type: "text/html"
    )
  end

  # コントローラーからViewComponentをレンダリングしていて、かつ
  # turbo-rails gemが読み込まれているときは `content_type` と `formats` の指定が必要
  # https://viewcomponent.org/guide/getting-started.html#rendering-from-controllers
  # https://github.com/ViewComponent/view_component/issues/1534#issuecomment-1986541830
  private def render_component(component, **)
    render(component, content_type: "text/html", formats: :html, **)
  end
end
