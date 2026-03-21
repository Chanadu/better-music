type EmptyStateOptions = {
	title: string;
	description: string;
	buttonLabel: string;
	onClick: () => void;
};

export const createEmptyState = ({
	title,
	description,
	buttonLabel,
	onClick,
}: EmptyStateOptions): HTMLDivElement => {
	const wrapper = document.createElement('div');
	wrapper.className =
		'rounded-[1.6rem] border border-border/60 bg-surface/85 px-5 py-8 text-center shadow-[0_20px_40px_-32px_rgba(15,23,42,0.4)] backdrop-blur-sm sm:px-7 sm:py-10';

	const titleElement = document.createElement('h2');
	titleElement.className = 'text-lg font-semibold text-text sm:text-xl';
	titleElement.textContent = title;
	wrapper.appendChild(titleElement);

	const descriptionElement = document.createElement('p');
	descriptionElement.className = 'mx-auto mt-2 max-w-md text-sm leading-6 text-text-muted';
	descriptionElement.textContent = description;
	wrapper.appendChild(descriptionElement);

	const button = document.createElement('button');
	button.type = 'button';
	button.className =
		'ui-pressable mt-5 inline-flex min-h-11 items-center justify-center rounded-xl bg-primary px-4 py-3 text-sm font-semibold text-primary-fg shadow-md shadow-primary/20 transition-all duration-300 hover:-translate-y-0.5 hover:bg-primary-hover hover:shadow-lg hover:shadow-primary/30 focus:ring-4 focus:ring-primary/20';
	button.textContent = buttonLabel;
	button.addEventListener('click', onClick);
	wrapper.appendChild(button);

	return wrapper;
};
