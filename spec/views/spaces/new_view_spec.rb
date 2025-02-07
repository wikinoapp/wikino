# typed: false
# frozen_string_literal: true

RSpec.describe Spaces::NewView, type: :view do
  it "スペース作成画面が表示されること" do
    Current.viewer = create(:user)

    render_inline(Spaces::NewView.new(form: NewSpaceForm.new))

    expect(page).to have_text("新規スペース")
  end
end
