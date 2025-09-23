# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe BaseUI::DialogComponent, type: :view do
  it "dialog要素を正しくレンダリングすること" do
    rendered = render_inline(BaseUI::DialogComponent.new(id: "test-dialog", class_name: "custom-class")) do
      "Test content"
    end

    expect(rendered.to_html).to include("<dialog")
    expect(rendered.to_html).to include('id="test-dialog"')
    expect(rendered.to_html).to include("custom-class")
    expect(rendered.to_html).to include("Test content")
  end
end