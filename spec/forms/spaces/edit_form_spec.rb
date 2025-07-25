# typed: false
# frozen_string_literal: true

RSpec.describe Spaces::EditForm, type: :form do
  it "Space が指定されていないとき、エラーになること" do
    form = Spaces::EditForm.new

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Space record can't be blank")
  end

  it "識別子が空文字列のとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Identifier can't be blank")
  end

  it "識別子が `nil` のとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Identifier can't be blank")
  end

  it "識別子が21文字のとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: "a" * 21)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Identifier is too long (maximum is 20 characters)")
  end

  it "識別子が予約語のとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: "www")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Identifier cannot be used")
  end

  it "識別子の形式が不正なとき、エラーになること" do
    form = Spaces::EditForm.new(identifier: "a@b")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Identifier is invalid")
  end

  it "識別子がすでに使われているとき、エラーになること" do
    create(:space_record, identifier: "a")
    space = create(:space_record)
    form = Spaces::EditForm.new(space_record: space, identifier: "a")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Identifier has already been taken")
  end

  it "名前が空文字列のとき、エラーになること" do
    form = Spaces::EditForm.new(name: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name can't be blank")
  end

  it "名前が `nil` のとき、エラーになること" do
    form = Spaces::EditForm.new(name: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name can't be blank")
  end

  it "名前が31文字のとき、エラーになること" do
    form = Spaces::EditForm.new(name: "a" * 31)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name is too long (maximum is 30 characters)")
  end
end