# typed: false
# frozen_string_literal: true

RSpec.describe Spaces::NewView, type: :view do
  it "スペース作成画面が表示されること" do
    Current.viewer = create(:user)

    render_inline(Spaces::NewView.new(
      current_user_entity: Current.viewer.user_entity,
      form: NewSpaceForm.new
    ))

    expect(page).to have_text("新規スペース")
  end
end
