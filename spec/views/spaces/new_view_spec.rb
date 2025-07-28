# typed: false
# frozen_string_literal: true

RSpec.describe Spaces::NewView, type: :view do
  it "スペース作成画面が表示されること" do
    user_record = create(:user_record)
    current_user = UserRepository.new.to_model(user_record:)

    render_inline(Spaces::NewView.new(
      current_user:,
      form: Spaces::CreationForm.new
    ))

    expect(page).to have_text("新規スペース")
  end
end
