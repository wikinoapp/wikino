# typed: false
# frozen_string_literal: true

RSpec.describe "トップページ", type: :system do
  it do
    visit "/"

    expect(page).to have_content "A Wiki app where you write in Markdown"
  end
end
